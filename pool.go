package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/porterr"
	"github.com/streadway/amqp"
	"sync"
	"time"
)

// RabbitMq server Pool
type ServerPool struct {
	// Connection pools
	pool map[string]*ConnectionPool
	// Mutex
	m sync.Mutex
	// logger
	logger gocli.Logger
}

// Init server pool
func NewServerPool(l gocli.Logger) *ServerPool {
	return &ServerPool{
		pool:   make(map[string]*ConnectionPool),
		logger: l,
	}
}

// Get connection pool
// If not - create
func (sp *ServerPool) GetConnectionPoolOrCreate(server string) *ConnectionPool {
	sp.m.Lock()
	defer sp.m.Unlock()
	if _, ok := sp.pool[server]; !ok {
		p := NewConnectionPool()
		go func() {
			// idle connections
			e := p.idle()
			// log error
			if e != nil {
				sp.logger.Errorln(e)
			}
		}()
		sp.pool[server] = p
	}
	return sp.pool[server]
}

// Connection pool
type ConnectionPool struct {
	// Connection pool
	// Uses round robin algorithm
	pool []*connection
	// cursor for current connection
	cursor int
	// Lock until using
	m sync.Mutex
	// flag shows that idle function in process
	fIdle bool
	// exit
	exit chan bool
	// request per second
	rps int
}

// Init connection pool
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{}
}

// Connection struct
type connection struct {
	// amqp connection
	conn *amqp.Connection
	// amqp channel
	channel *amqp.Channel
	// idle deadline UnixNano
	deadline int64
	// is unused
	remove bool
}

// Publish message
func (c *connection) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (e porterr.IError) {
	// channel publish
	err := c.channel.Publish(exchange, key, mandatory, immediate, msg)
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "exchange publish: %s", err.Error())
	}
	return
}

// Init idle worker fo pool
func (cp *ConnectionPool) idle() (e porterr.IError) {
	if cp.fIdle {
		return porterr.New(porterr.PortErrorProducer, "idle already in process")
	}
	cp.fIdle = true
	defer func() {
		cp.fIdle = false
	}()
	// cursor
	var cursor int
	// connection
	var conn *connection
	// current unix nano time
	var now int64
	for {
		select {
		case <-cp.exit:
			return
		default:
		}
		now = time.Now().UnixNano()
		for cursor, conn = range cp.pool {
			if now > conn.deadline && !conn.remove {
				conn.remove = true
			}
			if conn.remove && (now > (conn.deadline + int64(time.Second*10))) {
				e = cp.closeConnection(cursor)
				// for reindex
				break
			}
		}
		// Sleep second before next round
		time.Sleep(time.Second)
		cp.rps = 0
	}
}

// Close connection
func (cp *ConnectionPool) closeConnection(cursor int) (e porterr.IError) {
	cp.m.Lock()
	defer cp.m.Unlock()
	err := cp.pool[cursor].channel.Close()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't close channel: %s", err.Error())
	}
	err = cp.pool[cursor].conn.Close()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't close connection: %s", err.Error())
	}
	// remove connection from pool
	cp.pool = append(cp.pool[:cursor], cp.pool[cursor+1:]...)
	return e
}

// Dial to rabbit mq
func (cp *ConnectionPool) dial(s RabbitServer) (e porterr.IError) {
	c := &connection{}
	var err error
	c.conn, err = amqp.Dial(s.String())
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't dial to RabbitMq server (%s): %s", s.String(), err.Error())
		return
	}
	c.channel, err = c.conn.Channel()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't get channel: %s", err.Error())
		return
	}
	// Set confirm mode
	if err := c.channel.Confirm(false); err != nil {
		return porterr.NewF(porterr.PortErrorProducer, "Confirm mode set failed: %s ", err.Error())
	}
	c.deadline = time.Now().Add(s.MaxIdleConnectionLifeTime).UnixNano()
	cp.pool = append(cp.pool, c)
	return
}

// Count how many connections must be removed from pool
func (cp *ConnectionPool) getRemovedCount() (count int) {
	for _, c := range cp.pool {
		if c.remove {
			count++
		}
	}
	return
}

// Get current connection using round robin algorithm
func (cp *ConnectionPool) GetConnection(s RabbitServer) (c *connection, e porterr.IError) {
	cp.m.Lock()
	defer cp.m.Unlock()
	if len(cp.pool) == 0 || len(cp.pool) == cp.getRemovedCount() {
		// dial first connection
		e = cp.dial(s)
		if e != nil {
			return
		}
	} else if len(cp.pool) < (cp.rps / DefaultMaxConnectionOnRPS * s.MaxConnections) && s.MaxConnections > len(cp.pool) {
		// dial until connections limit
		e = cp.dial(s)
		if e != nil {
			return
		}
	}
	for {
		if cp.cursor > len(cp.pool)-1 {
			cp.cursor = 0
		}
		c = cp.pool[cp.cursor]
		if c.remove {
			cp.cursor++
			continue
		}
		// update deadline
		c.deadline = time.Now().Add(s.MaxIdleConnectionLifeTime).UnixNano()
		cp.cursor++
		break
	}
	// increase rps counter
	cp.rps++
	return
}

// Publish message to queue
func (cp *ConnectionPool) Publish(p amqp.Publishing, s RabbitServer, q RabbitQueue, route ...string) (e porterr.IError) {
	// Get connection with initiated channel
	conn, e := cp.GetConnection(s)
	if e != nil {
		return
	}
	// Publish to all routing keys
	for _, key := range route {
		err := conn.Publish(q.Exchange, key, false, false, p)
		if err != nil {
			conn.remove = true
			e = porterr.NewF(porterr.PortErrorProducer, "exchange publish: %s", err.Error())
			break
		}
	}
	return
}
