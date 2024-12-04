package integration

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	resp.Body.Close()
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
	cmd := exec.Command(bin)

	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_PUBADDR=%s", pubAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_APIADDR=%s", apiAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_ERROR_LOG=%s", tempLogfile.Name()))
	cmd.Env = append(cmd.Env, "CONTENT_STORE_DATABASE_URL="+postgresContainer.MustConnectionString(context.Background()))
	cmd.Env = append(cmd.Env, extraEnv...)

	if os.Getenv("ROUTER_DEBUG_TESTS") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	waitForServerUp(pubAddr)

	runningRouters[port] = cmd
	return nil
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

func waitForServerUp(addr string) {
	for i := 0; i < 20; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	panic("Server not accepting connections after 20 attempts")
}
