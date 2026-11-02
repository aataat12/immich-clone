package web

import (
	"net"
	"sync"
)

// MaxConnections bounds accepted sockets, including idle keep-alive connections.
// This single-user loopback service normally needs far fewer connections.
const MaxConnections = 64

// LimitConnections closes excess accepted sockets before an HTTP goroutine or
// request body is allocated. It does not create a waiting admission queue.
func LimitConnections(listener net.Listener, limit int) net.Listener {
	if limit < 1 {
		panic("positive connection limit required")
	}
	return &limitedListener{Listener: listener, slots: make(chan struct{}, limit)}
}

type limitedListener struct {
	net.Listener
	slots chan struct{}
}

func (l *limitedListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		select {
		case l.slots <- struct{}{}:
			return &limitedConnection{Conn: conn, release: func() { <-l.slots }}, nil
		default:
			_ = conn.Close()
		}
	}
}

type limitedConnection struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *limitedConnection) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}
