package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	// revive:disable:dot-imports
	. "github.com/onsi/gomega"
	// revive:enable:dot-imports
)

const (
	routerPort = 3169
	apiPort    = 3168
)

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

	defaultBackend := startDummyBackend("dummy-default-backend", 404)

	bin := os.Getenv("BINARY")
	if bin == "" {
		bin = "../router"
	}
	cmd := exec.Command(bin)

	cmd.Env = append(cmd.Environ(), "ROUTER_MONGO_DB=router_test")
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_PUBADDR=%s", pubAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_APIADDR=%s", apiAddr))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_DEFAULT_BACKEND_URL=%s", defaultBackend.URL))
	cmd.Env = append(cmd.Env, fmt.Sprintf("ROUTER_ERROR_LOG=%s", tempLogfile.Name()))
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
