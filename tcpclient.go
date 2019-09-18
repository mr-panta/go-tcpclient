package tcpclient

import (
	"errors"
	"io/ioutil"
	"net"
	"sync"
	"time"
)

// Client contains TCP connection pool and provides
// APIs for communicating via TCP connection
type Client interface {
	// Send is used to send and get TCP data via TCP connection
	Send(input []byte) (output []byte, err error)
	// Close is used to close all connections in connection pool
	Close() (err error)
}

type defaultClient struct {
	addr            string
	minConns        int
	maxConns        int
	idleConnTimeout time.Duration
	waitConnTimeout time.Duration
	clearPeriod     time.Duration
	poolSize        int
	poolLock        sync.Mutex
	connPool        chan *connection
}

type connection struct {
	tcpConn    net.Conn
	lastActive time.Time
}

// NewClient is used to create TCP client.
// `addr` is TCP host address.
// `minConns` is the minimum number of connections in connection pool.
// `maxConns` is the maximum number of connections in connection pool.
// `idleConnTimeout` is the duration that allows connection to stay idle.
// `waitConnTimeout` is the duration that allows connection pool to stay empty,
// otherwise new connection will be created.
// The connection pool manager will clear idle connections every `clearPeriod` duration.
func NewClient(addr string, minConns, maxConns int, idleConnTimeout, waitConnTimeout,
	clearPeriod time.Duration) (client Client, err error) {

	c := &defaultClient{
		addr:            addr,
		minConns:        minConns,
		maxConns:        maxConns,
		idleConnTimeout: idleConnTimeout,
		waitConnTimeout: waitConnTimeout,
		clearPeriod:     clearPeriod,
		poolSize:        0,
		poolLock:        sync.Mutex{},
		connPool:        make(chan *connection, maxConns),
	}
	for i := 0; i < c.minConns; i++ {
		if _, err = c.fillConnPool(false); err != nil {
			c.Close()
			return client, err
		}
	}
	go c.poolManager()
	return c, nil
}

// Send is used to send and get TCP data via TCP connection
func (c *defaultClient) Send(input []byte) (output []byte, err error) {
	if c.poolSize == 0 {
		return nil, errors.New("all connections in connection pool are already closed")
	}
	conn := &connection{}
	select {
	case conn = <-c.connPool:
	case <-time.After(c.waitConnTimeout):
		conn, err = c.fillConnPool(true)
		if err != nil {
			return nil, err
		}
	}
	conn.lastActive = time.Now()
	defer func() {
		c.connPool <- conn
	}()
	_, err = conn.tcpConn.Write(input)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(conn.tcpConn)
}

// Close is used to close all connections in connection pool
func (c *defaultClient) Close() (err error) {
	for empty := false; !empty; {
		empty, err = c.drainConnPool(nil, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *defaultClient) fillConnPool(getConn bool) (conn *connection, err error) {
	c.poolLock.Lock()
	defer c.poolLock.Unlock()
	if c.poolSize == c.maxConns {
		return nil, errors.New("connection pool is full")
	}
	c.poolSize++
	tcpConn, err := net.Dial("tcp", c.addr)
	if err != nil {
		return nil, err
	}
	conn = &connection{
		tcpConn:    tcpConn,
		lastActive: time.Now(),
	}
	if getConn {
		return conn, nil
	}
	c.connPool <- conn
	return nil, nil
}

func (c *defaultClient) poolManager() {
	for {
		if c.poolSize == 0 {
			return
		}
		conn := <-c.connPool
		if time.Since(conn.lastActive) > c.idleConnTimeout {
			_, _ = c.drainConnPool(conn, false)
		} else {
			c.connPool <- conn
		}
		time.Sleep(c.clearPeriod)
	}
}

func (c *defaultClient) drainConnPool(conn *connection, forceMode bool) (empty bool, err error) {
	c.poolLock.Lock()
	defer c.poolLock.Unlock()
	if c.poolSize == 0 {
		return true, errors.New("connection pool is empty")
	}
	if conn == nil {
		conn = <-c.connPool
	}
	defer func() {
		if err != nil {
			c.connPool <- conn
		}
	}()
	if c.poolSize == c.minConns && !forceMode {
		err = errors.New("pool size cannot be lower than minimum number of connections")
		return false, err
	}
	c.poolSize--
	err = conn.tcpConn.Close()
	if err != nil {
		return false, err
	}
	empty = c.poolSize == 0
	return empty, nil
}
