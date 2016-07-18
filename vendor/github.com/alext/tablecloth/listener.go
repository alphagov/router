package tablecloth

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"syscall"
	"time"
)

type watchedConn struct {
	net.Conn
	listener *gracefulListener
}

func (c *watchedConn) Close() error {
	err := c.Conn.Close()
	c.listener.decCount()
	return err
}

func resumeOrListen(fd int, addr string) (*gracefulListener, error) {
	var l net.Listener
	var err error
	if fd != 0 {
		f := os.NewFile(uintptr(fd), "listen socket")
		l, err = net.FileListener(f)
		e := f.Close()
		if e != nil {
			return nil, e
		}
	} else {
		l, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return nil, err
	}

	return &gracefulListener{Listener: l}, nil
}

type gracefulListener struct {
	net.Listener
	connCount int64
	stopping  bool
}

func (l *gracefulListener) Addr() net.Addr {
	tcpListener, ok := l.Listener.(*net.TCPListener)
	if ok {
		return tcpListener.Addr()
	}
	return nil
}

func (l *gracefulListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	c = &watchedConn{Conn: c, listener: l}
	l.incCount()
	return c, nil
}

func (l *gracefulListener) Close() error {
	l.stopping = true
	return l.Listener.Close()
}

func (l *gracefulListener) getCount() int64 {
	return atomic.LoadInt64(&l.connCount)
}
func (l *gracefulListener) incCount() {
	atomic.AddInt64(&l.connCount, 1)
}
func (l *gracefulListener) decCount() {
	atomic.AddInt64(&l.connCount, -1)
}

func (l *gracefulListener) waitForClients(timeout time.Duration) error {
	if l.getCount() == 0 {
		return nil
	}
	timeoutCh := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if l.getCount() == 0 {
				return nil
			}
		case <-timeoutCh:
			return fmt.Errorf("Still %d active clients after %s", l.getCount(), timeout)
		}
	}
}

func (l *gracefulListener) prepareFd() (fd int, err error) {
	tl := l.Listener.(*net.TCPListener)
	fl, err := tl.File()
	if err != nil {
		return 0, err
	}
	defer fl.Close()

	// The TCPListener.File() sets the underlying socket to be blocking
	// (http://git.io/veIh6).  This alters the behaviour of Accept such that
	// when the listener fd is closed, Accept doesn't return an error until the
	// next connection comes in.
	//
	// Setting this back to non-blocking allows this to continue to use the
	// epoll mechanism meaning that Accept will return an error immediately
	// when the listener fd is closed.
	syscall.SetNonblock(int(fl.Fd()), true)

	// Dup the fd to clear the CloseOnExec flag
	fd, err = syscall.Dup(int(fl.Fd()))
	if err != nil {
		return 0, err
	}
	return fd, nil
}
