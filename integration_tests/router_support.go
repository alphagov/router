package integration

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var runningRouters = make(map[int]*exec.Cmd)

func startRouter(port, apiPort int) error {
	pubaddr := fmt.Sprintf(":%d", port)
	apiaddr := fmt.Sprintf(":%d", apiPort)

	cmd := exec.Command("../router")

	env := newEnvMap(os.Environ())
	env["ROUTER_PUBADDR"] = pubaddr
	env["ROUTER_APIADDR"] = apiaddr
	env["ROUTER_MONGO_DB"] = "router_test"
	env["ROUTER_ERROR_LOG"] = "/dev/null"
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
