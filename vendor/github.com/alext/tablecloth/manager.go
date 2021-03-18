package tablecloth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// How long to wait for a newly started process to start serving requests.
var StartupDelay = 5 * time.Second

// The maximum time to wait for outstanding connections to complete after
// closing the servers.
var CloseWaitTimeout = 30 * time.Second

// Optional: the working directory for the application.  This directory (if specified)
// will be changed to before re-execing.
//
// This is typically used when the working directory is accessed via a symlink
// so that the symlink is re-evaluated when re-execing. This allows updating a symlink
// to point at a new version of the application, and for this to be picked up.
var WorkingDir string

var (
	theManager = &manager{}
	// variable indirection to facilitate testing
	setupFunc = theManager.setup
)

/*
ListenAndServe wraps the equivelent function from net/http, and therefore behaves in
the same way.  It adds the necessary tracking for the connections created so that
they can be passed to new processes etc.

If using more than one call to ListenAndServe in an application, each call must pass
a unique string as identifier.  This is used to identify the file descriptors passed
to new processes.  If identifier is not specified, it uses a value of "default".

In order for the seamless restarts to work it is important that the calling application
exits after all ListenAndServe calls have returned.

A simple example:

package main

	import (
		"fmt"
		"net/http"

		"github.com/alext/tablecloth"
	)

	func main() {
		tablecloth.ListenAndServe(":8080", http.HandlerFunc(handler))
	}

	func handler(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello world")
	}

A more involved example that uses multiple ports:

	package main

	import (
		"fmt"
		"net/http"
		"sync"

		"github.com/alext/tablecloth"
	)

	func main() {
		wg := &sync.WaitGroup{}
		wg.Add(2)
		go serve(":8080", "main", wg)
		go serve(":8081", "admin", wg)
		wg.Wait()
	}

	func serve(listenAddr, ident string, wg *sync.WaitGroup) {
		defer wg.Done()
		tablecloth.ListenAndServe(listenAddr, http.HandlerFunc(handler), ident)
	}

	func handler(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello world")
	}
*/
func ListenAndServe(addr string, handler http.Handler, identifier ...string) error {
	theManager.once.Do(setupFunc)

	ident := "default"
	if len(identifier) >= 1 {
		ident = identifier[0]
	}

	return theManager.listenAndServe(addr, handler, ident)
}

type serverInfo struct {
	listener *net.TCPListener
	server   *http.Server
	wg       sync.WaitGroup
}

type manager struct {
	once          sync.Once
	servers       map[string]*serverInfo
	serversLock   sync.Mutex
	activeServers sync.WaitGroup
	inParent      bool
}

func (m *manager) setup() {
	m.servers = make(map[string]*serverInfo)
	m.inParent = os.Getenv("TEMPORARY_CHILD") != "1"

	go m.handleSignals()

	if m.inParent {
		go m.stopTemporaryChild()
	}
}

func (m *manager) listenAndServe(addr string, handler http.Handler, ident string) error {
	m.activeServers.Add(1)
	defer m.activeServers.Done()

	si, err := m.setupServer(addr, ident, handler)
	if err != nil {
		return err
	}

	si.wg.Add(1)
	err = si.server.Serve(si.listener)
	if err == http.ErrServerClosed {
		si.wg.Wait() // Done() called in stopServers()
		if m.inParent {
			// This function will now never return, so the above defer won't happen.
			m.activeServers.Done()

			// prevent this goroutine returning before the server has re-exec'd
			// This is to cover the case where this is the main goroutine, and exiting
			// would therefore prevent the re-exec happening
			c := make(chan bool)
			<-c
		}
		return nil
	}
	return err
}

func (m *manager) setupServer(addr, ident string, handler http.Handler) (*serverInfo, error) {
	m.serversLock.Lock()
	defer m.serversLock.Unlock()

	if m.servers[ident] != nil {
		return nil, errors.New("duplicate ident")
	}

	l, err := resumeOrListen(listenFdFromEnv(ident), addr)
	if err != nil {
		return nil, err
	}
	si := &serverInfo{
		listener: l,
		server:   &http.Server{Handler: handler},
	}
	m.servers[ident] = si
	return si, nil
}

func listenFdFromEnv(ident string) int {
	listenFD, err := strconv.Atoi(os.Getenv("LISTEN_FD_" + ident))
	if err != nil {
		return 0
	}
	return listenFD
}

func (m *manager) handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	for _ = range c {
		m.handleHUP()
	}
}

func (m *manager) handleHUP() {
	m.serversLock.Lock()
	defer m.serversLock.Unlock()

	if m.inParent {
		err := m.upgradeServer()
		if err != nil {
			log.Println("[tablecloth] error starting new server, aborting reload:", err)
			return
		}
	}

	m.stopServers()
}

func (m *manager) upgradeServer() error {
	fds := make(map[string]int, len(m.servers))
	for ident, si := range m.servers {
		fd, err := prepareListenerFd(si.listener)
		if err != nil {
			// Close any that were successfully prepared so we don't leak.
			closeFds(fds)
			return err
		}
		fds[ident] = fd
	}

	proc, err := m.startTemporaryChild(fds)
	if err != nil {
		// Close all the copied file descriptors so we don't leak.
		closeFds(fds)
		return err
	}

	time.Sleep(StartupDelay)

	err = assertChildStillRunning(proc.Pid)
	if err != nil {
		closeFds(fds)
		return err
	}

	go m.reExecSelf(fds, proc.Pid)
	return nil
}

func closeFds(fds map[string]int) {
	for ident, fd := range fds {
		os.NewFile(uintptr(fd), ident).Close()
	}
}

func assertChildStillRunning(pid int) error {
	pid, err := syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
	if err != nil {
		return fmt.Errorf("wait4 error: %s", err.Error())
	}
	if pid != 0 {
		return fmt.Errorf("child no longer running after StartupDelay(%s)", StartupDelay)
	}
	return nil
}

func (m *manager) stopServers() {
	ctx, _ := context.WithTimeout(context.Background(), CloseWaitTimeout)
	for _, si := range m.servers {
		go func(si *serverInfo) {
			defer si.wg.Done()
			err := si.server.Shutdown(ctx)
			if err != nil {
				log.Println("[tablecloth] error shutting down server:", err)
			}
		}(si)
	}
}

func (m *manager) reExecSelf(fds map[string]int, childPid int) {
	// wait until there are no active servers
	m.activeServers.Wait()

	em := newEnvMap(os.Environ())
	for ident, fd := range fds {
		em["LISTEN_FD_"+ident] = strconv.Itoa(fd)
	}
	em["TEMPORARY_CHILD_PID"] = strconv.Itoa(childPid)

	if WorkingDir != "" {
		os.Chdir(WorkingDir)
	}
	syscall.Exec(os.Args[0], os.Args, em.ToEnv())
}

func (m *manager) startTemporaryChild(fds map[string]int) (proc *os.Process, err error) {

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	em := newEnvMap(os.Environ())
	for ident, fd := range fds {
		em["LISTEN_FD_"+ident] = strconv.Itoa(fd)
	}
	em["TEMPORARY_CHILD"] = "1"
	cmd.Env = em.ToEnv()
	if WorkingDir != "" {
		cmd.Dir = WorkingDir
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd.Process, nil
}

func (m *manager) stopTemporaryChild() {
	childPid, err := strconv.Atoi(os.Getenv("TEMPORARY_CHILD_PID"))
	if err != nil {
		// non-integer/blank TEMPORARY_CHILD_PID so ignore
		return
	}

	time.Sleep(StartupDelay)

	proc, err := os.FindProcess(childPid)
	if err != nil {
		//TODO: something better here?
		// Failed to find process
		return
	}
	err = proc.Signal(syscall.SIGHUP)
	if err != nil {
		//TODO: better error handling
		return
	}
	_, err = proc.Wait()
	if err != nil {
		//TODO: better error handling
		return
	}
}
