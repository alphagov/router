package integration

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	// revive:disable:dot-imports
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
)

const (
	routerPort = 3169
	apiPort    = 3168
)

func runPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, func(), error) {
	dbName := "content_store"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:14-alpine",
		postgres.WithInitScripts(filepath.Join("testdata", "init-user-db.sh")),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to run container: %w", err)
	}

	return postgresContainer, func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}, nil
}

func routerURL(port int, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
}

// Send a reload request to Router's API server
func reloadRoutes(port int) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/reload", port),
		http.NoBody,
	)
	Expect(err).NotTo(HaveOccurred())

	resp, err := http.DefaultClient.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(202))
	_ = resp.Body.Close()
	// Now that reloading is done asynchronously, we need a small sleep to ensure
	// it has actually been performed.
	time.Sleep(time.Millisecond * 50)
}

var runningRouters = make(map[int]*exec.Cmd)

func startRouter(port, apiPort int, extraEnv []string) error {
	host := "localhost"
	pubAddr := net.JoinHostPort(host, strconv.Itoa(port))
	apiAddr := net.JoinHostPort(host, strconv.Itoa(apiPort))

	bin := os.Getenv("BINARY")
	if bin == "" {
		bin = "../router"
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, bin) //gosec:disable G204 //gosec:disable G702-- We intentionally want to exec a sub process with a var

	coverageDir, err := getCoverageDir()
	if err != nil {
		return err
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("GOCOVERDIR=%s", coverageDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_PUBADDR=%s", pubAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_APIADDR=%s", apiAddr))
	cmd.Env = append(cmd.Env, "CONTENT_STORE_DATABASE_URL="+postgresContainer.MustConnectionString(context.Background()))
	cmd.Env = append(cmd.Env, extraEnv...)

	if os.Getenv("ROUTER_DEBUG_TESTS") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	waitForServerUp(ctx, pubAddr)

	runningRouters[port] = cmd
	return nil
}

func sendSignalToRouter(port int, sig os.Signal) (*exec.Cmd, error) {
	router, ok := runningRouters[port]
	if !ok {
		return nil, fmt.Errorf("no running router on on port %d", port)
	}

	err := router.Process.Signal(sig)
	if err != nil {
		return nil, err
	}

	return router, nil
}

func ensureRouterTerminated(port int) error {
	cmd, ok := runningRouters[port]
	if !ok {
		// Router has already been terminated and cleaned up, or was never created
		return nil
	}

	if cmd.ProcessState != nil {
		// The process already terminated, just cleanup
		delete(runningRouters, port)
		return nil
	}

	err := cmd.Process.Signal(syscall.SIGINT)
	if errors.Is(err, os.ErrProcessDone) {
		delete(runningRouters, port)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to send termination signal to running router process, error: %w", err)
	}

	state, err := cmd.Process.Wait()
	waitStatus, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return fmt.Errorf("could not convert process wait status to a syscall.WaitStatus, maybe you're not on a *nix platform?")
	}

	switch {
	// On *nix if a process exits because of a signal it is not 'exited', instead it is 'signalled', either way we shut down fine
	case state.Exited() || waitStatus.Signaled():
		delete(runningRouters, port)
		return nil
	case err != nil:
		return fmt.Errorf("running router process did not exit with error: %w", err)
	default:
		return fmt.Errorf(
			"the running router process both did not exit, and did not return an error waiting for termination! The state was %s",
			state,
		)
	}
}

func getCoverageDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Abs(filepath.Clean(path.Join(cwd, "..", "coverage", "integration")))
}

func stopRouter(port int) {
	cmd := runningRouters[port]
	if cmd != nil && cmd.Process != nil {
		err := cmd.Process.Signal(syscall.SIGINT)
		Expect(err).NotTo(HaveOccurred())
		_, err = cmd.Process.Wait()
		Expect(err).NotTo(HaveOccurred())
	}
	delete(runningRouters, port)
}

func waitForServerUp(ctx context.Context, addr string) {
	for i := 0; i < 20; i++ {
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	panic("Server not accepting connections after 20 attempts")
}
