package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"net/http"
)

// Get server
func (c *Config) GetServer(name string) (*RabbitServer, gocli.IError) {
	server, ok := c.Servers[name]
	if !ok {
		return nil, gocli.NewError(fmt.Sprintf("server %s not found in rabbit config", name), http.StatusInternalServerError)
	}
	if server.Vhost == "" {
		return nil, gocli.NewError(fmt.Sprintf("vhost is incorrect for server %s in rabbit config", name), http.StatusInternalServerError)
	}
	return &server, nil
}

// Get queue
func (c *Config) GetQueue(name string) (*RabbitQueue, gocli.IError) {
	queue, ok := c.Queues[name]
	if !ok {
		return nil, gocli.NewError(fmt.Sprintf("queue %s not found in rabbit config", name), http.StatusInternalServerError)
	}
	queue.Name = name
	return &queue, nil
}

// Get connection string
func (srv *RabbitServer) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%v/%s", srv.User, srv.Password, srv.Host, srv.Port, srv.Vhost)
}

// New rabbit Application
func NewApplication(config Config, app gocli.Application, registry Registry) *Application {
	return &Application{
		config:   config,
		base:     app,
		registry: registry,
	}
}

// On Error
func (a *Application) onError(err error, msg string) {
	if err != nil {
		a.base.GetLogger(gocli.LogLevelDebug).Errorf("%s: %s", msg, err)
		a.base.FatalError(fmt.Errorf("%s: %s", msg, err))
	}
}
