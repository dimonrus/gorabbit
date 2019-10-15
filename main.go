package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/porterr"
)

// Get server
func (c *Config) GetServer(name string) (*RabbitServer, porterr.IError) {
	server, ok := c.Servers[name]
	if !ok {
		return nil, porterr.NewF(porterr.PortErrorSystem, "server %s not found in rabbit config", name)
	}
	if server.Vhost == "" {
		return nil, porterr.NewF(porterr.PortErrorSystem, "vhost is incorrect for server %s in rabbit config", name)
	}
	return &server, nil
}

// Get queue
func (c *Config) GetQueue(name string) (*RabbitQueue, porterr.IError) {
	queue, ok := c.Queues[name]
	if !ok {
		return nil, porterr.NewF(porterr.PortErrorSystem, "queue %s not found in rabbit config", name)
	}
	queue.Name = name
	return &queue, nil
}

// Get connection string
func (srv *RabbitServer) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%v/%s", srv.User, srv.Password, srv.Host, srv.Port, srv.Vhost)
}

// New rabbit Application
func NewApplication(config Config, app gocli.Application) *Application {
	return &Application{
		config: config,
		base:   app,
	}
}

// On Error
func (a *Application) onError(err error, msg string) {
	if err != nil {
		a.base.GetLogger(gocli.LogLevelDebug).Errorf("%s: %s", msg, err)
		a.base.FatalError(fmt.Errorf("%s: %s", msg, err))
	}
}
