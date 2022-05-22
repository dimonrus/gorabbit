package gorabbit

import (
	"github.com/dimonrus/gocli"
	"testing"
	"time"
)

func TestServerPool_GetConnectionPoolOrCreate(t *testing.T) {
	pool := NewServerPool(gocli.NewLogger(gocli.LoggerConfig{}))
	for i := 1; i < 5; i++ {
		go func() {
			conn := pool.GetConnectionPoolOrCreate("local")
			_ = conn
		}()
	}
	time.Sleep(time.Second)
}
