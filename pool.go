package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gohelp"
	"github.com/dimonrus/porterr"
	amqp "github.com/rabbitmq/amqp091-go"
	"sync"
	"sync/atomic"
	"time"
)

// MaxMessagesPerConnection will close connection on reach limit
const MaxMessagesPerConnection = int64(50000)

// ServerPool RabbitMq server Pool
type ServerPool struct {
	// Connection pools
	pool map[string]*ConnectionPool
	// Mutex
	m sync.Mutex
	// logger
	logger gocli.Logger
}

// NewServerPool Init server pool
func NewServerPool(l gocli.Logger) *ServerPool {
	return &ServerPool{
		pool:   make(map[string]*ConnectionPool),
		logger: l,
	}
}

// GetConnectionPoolOrCreate Get connection pool
// If not - create
func (sp *ServerPool) GetConnectionPoolOrCreate(server string, maxConnections int) *ConnectionPool {
	sp.m.Lock()
	defer sp.m.Unlock()
	if _, ok := sp.pool[server]; !ok {
		p := NewConnectionPool(maxConnections)
		go func(pool *ConnectionPool) {
			// idle connections
			e := pool.idle(sp.logger)
			// log error
			if e != nil {
				sp.logger.Errorln(e)
			}
		}(p)
		sp.pool[server] = p
	}
	return sp.pool[server]
}

// ConnectionPool Connection pool
type ConnectionPool struct {
	// Connection pool
	// Uses round robin algorithm
	pool []*connection
	// cursor for current connection
	cursor int32
	// Lock until using
	m sync.Mutex
	// flag shows that idle function in process
	fIdle bool
	// exit
	exit chan struct{}
	// request per second
	rps int32
}

// NewConnectionPool Init connection pool
func NewConnectionPool(maxConnection int) *ConnectionPool {
	return &ConnectionPool{
		pool: make([]*connection, maxConnection),
	}
}

// Connection struct
type connection struct {
	// amqp connection
	conn *amqp.Connection
	// amqp channel
	channel *amqp.Channel
	// limit for message rate
	limitRate int64
	// idle deadline UnixNano
	deadline int64
	// 0 - when connection is not busy
	busy int32
}

// IsBusy check if connection is busy
func (c *connection) IsBusy() bool {
	return atomic.LoadInt32(&c.busy) != 0
}

// Publish message
func (c *connection) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (e porterr.IError) {
	atomic.StoreInt32(&c.busy, 1)
	defer func() {
		atomic.AddInt64(&c.limitRate, 1)
		atomic.StoreInt32(&c.busy, 0)
	}()
	// channel publish
	err := c.channel.Publish(exchange, key, mandatory, immediate, msg)
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, err.Error())
	}
	return
}

// Init idle worker fo pool
func (cp *ConnectionPool) idle(logger gocli.Logger) (e porterr.IError) {
	if cp.fIdle {
		return porterr.New(porterr.PortErrorProducer, "idle already in process")
	}
	cp.fIdle = true
	var i int
	for {
		select {
		case <-cp.exit:
			cp.fIdle = false
			return
		default:
		}
		cp.m.Lock()
		if cp.pool[i] != nil && atomic.LoadInt32(&cp.pool[i].busy) == 0 {
			if atomic.LoadInt64(&cp.pool[i].limitRate)-MaxMessagesPerConnection >= 0 {
				e = cp.reopenChannel(i)
				if e != nil {
					logger.Errorln(e.Error())
				}
			} else if atomic.LoadInt64(&cp.pool[i].deadline) < time.Now().UnixNano() {
				e = cp.closeConnection(i)
				if e != nil {
					logger.Errorln(e.Error())
				}
			}
		}
		i++
		if i == len(cp.pool) {
			i = 0
		}
		cp.m.Unlock()
		time.Sleep(time.Millisecond * 100)
	}
}

// Close connection
func (cp *ConnectionPool) reopenChannel(cursor int) (e porterr.IError) {
	err := cp.pool[cursor].channel.Close()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't close channel: %s", err.Error())
	}
	cp.pool[cursor].channel, err = cp.pool[cursor].conn.Channel()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't open channel: %s", err.Error())
	}
	atomic.StoreInt64(&cp.pool[cursor].limitRate, 0)
	return e
}

// Close connection
func (cp *ConnectionPool) closeConnection(cursor int) (e porterr.IError) {
	err := cp.pool[cursor].channel.Close()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't close channel: %s", err.Error())
	}
	err = cp.pool[cursor].conn.Close()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Can't close connection: %s", err.Error())
	}
	cp.pool[cursor] = nil
	return e
}

// Dial to rabbit mq
func (cp *ConnectionPool) dial(s RabbitServer) (c *connection, e porterr.IError) {
	c = &connection{
		deadline: time.Now().Add(s.MaxIdleConnectionLifeTime).UnixNano(),
	}
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
	if err = c.channel.Confirm(false); err != nil {
		e = porterr.NewF(porterr.PortErrorProducer, "Confirm mode set failed: %s ", err.Error())
	}
	return
}

// GetConnection Get current connection using round robin algorithm
func (cp *ConnectionPool) GetConnection(s RabbitServer) (c *connection, e porterr.IError) {
	cp.m.Lock()
	defer cp.m.Unlock()
	var i = gohelp.GetRndNumber(0, s.MaxConnections)
	for {
		if cp.pool[i] != nil {
			if !cp.pool[i].IsBusy() && !cp.pool[i].conn.IsClosed() && !cp.pool[i].channel.IsClosed() {
				c = cp.pool[i]
				return
			} else if cp.pool[i].conn.IsClosed() || cp.pool[i].channel.IsClosed() {
				cp.pool[i], e = cp.dial(s)
				c = cp.pool[i]
				return
			}
		} else {
			cp.pool[i], e = cp.dial(s)
			c = cp.pool[i]
			return
		}
		i++
		if i == len(cp.pool) {
			i = 0
		}
	}
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
			e = porterr.NewF(porterr.PortErrorProducer, err.Error())
			break
		}
	}
	return
}
