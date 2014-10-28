package integration

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	. "github.com/onsi/gomega"
)

func routerURL(path string, optionalPort ...int) string {
	port := 3169
	if len(optionalPort) > 0 {
		port = optionalPort[0]
	}
	return fmt.Sprintf("http://localhost:%d%s", port, path)
}

func reloadRoutes(optionalPort ...int) {
	port := 3168
	if len(optionalPort) > 0 {
		port = optionalPort[0]
	}
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/reload", port), "", nil)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(200))
}

var runningRouters = make(map[int]*exec.Cmd)

func startRouter(port, apiPort int, optionalExtraEnv ...envMap) error {
	pubaddr := fmt.Sprintf(":%d", port)
	apiaddr := fmt.Sprintf(":%d", apiPort)

	cmd := exec.Command("../router")

	env := newEnvMap(os.Environ())
	env["ROUTER_PUBADDR"] = pubaddr
	env["ROUTER_APIADDR"] = apiaddr
	env["ROUTER_MONGO_DB"] = "router_test"
	env["ROUTER_ERROR_LOG"] = tempLogfile.Name()
	if len(optionalExtraEnv) > 0 {
		for k, v := range optionalExtraEnv[0] {
			env[k] = v
		}
	}
	cmd.Env = env.ToEnv()

	if os.Getenv("DEBUG_ROUTER") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	waitForServerUp(pubaddr)

	runningRouters[port] = cmd
	return nil
}

func stopRouter(port int) {
	cmd := runningRouters[port]
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGINT)
		cmd.Process.Wait()
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
