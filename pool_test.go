package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gohelp"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServerPool_GetConnectionPoolOrCreate(t *testing.T) {
	pool := NewServerPool(gocli.NewLogger(gocli.LoggerConfig{}))
	for i := 1; i < 5; i++ {
		go func() {
			conn := pool.GetConnectionPoolOrCreate("local", 10)
			_ = conn
		}()
	}
	time.Sleep(time.Second)
}

type conn struct {
	some     string
	deadline int64
	busy     int32
}

func (c *conn) Process(i int) {
	atomic.StoreInt32(&c.busy, 1)
	defer func() {
		atomic.StoreInt64(&c.deadline, time.Now().Add(time.Second*10).UnixNano())
		atomic.StoreInt32(&c.busy, 0)
	}()
	time.Sleep(time.Millisecond * 500)
	fmt.Println("Test", i)
}

func NewConnection() *conn {
	return &conn{
		some:     gohelp.RandString(10),
		deadline: time.Now().Add(time.Second * 5).UnixNano(),
	}
}

type Pool struct {
	// Connection pool
	pool []*conn
	// cursor for current connection
	cursor int32
	// Lock until using
	m sync.Mutex
	// flag shows that idle function in process
	// exit
	exit chan struct{}
	// request per second
	rps int32
}

func (p *Pool) GetConnection() *conn {
	p.m.Lock()
	defer p.m.Unlock()
	var i int
	for {
		if p.pool[i] != nil {
			if atomic.LoadInt32(&p.pool[i].busy) == 0 {
				return p.pool[i]
			}
		} else {
			p.pool[i] = NewConnection()
		}
		i++
		if i == len(p.pool) {
			i = 0
		}
	}
}

func (p *Pool) Idle() {
	var i int
	for {
		select {
		case <-p.exit:
			return
		default:
		}
		now := time.Now().UnixNano()
		p.m.Lock()
		if p.pool[i] != nil {
			if atomic.LoadInt64(&p.pool[i].deadline) < now && atomic.LoadInt32(&p.pool[i].busy) == 0 {
				p.pool[i] = nil
				fmt.Println("remove connection", i)
			}
		}
		i++
		if i == len(p.pool) {
			i = 0
		}
		p.m.Unlock()
		time.Sleep(time.Second)
	}
}

func NewPool() *Pool {
	p := &Pool{pool: make([]*conn, DefaultMaxConnections)}
	go func() {
		p.Idle()
	}()
	return p
}

func TestNewPool(t *testing.T) {
	c := make(chan struct{}, 0)
	p := NewPool()
	for i := 0; i < 100; i++ {
		go p.GetConnection().Process(i)
		go p.GetConnection().Process(i)
		go p.GetConnection().Process(i)
	}
	time.Sleep(time.Second * 11)
	for i := 0; i < 100; i++ {
		go p.GetConnection().Process(i)
		go p.GetConnection().Process(i)
		go p.GetConnection().Process(i)
	}
	<-c
}
