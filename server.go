package gorabbit

import (
	"fmt"
	"time"
)

const (
	// DefaultMaxConnections Default value for max conn
	DefaultMaxConnections = 10
	// DefaultMaxIdleConnectionLifeTime Default value for max idle lifetime
	DefaultMaxIdleConnectionLifeTime = 10 * time.Second
	// DefaultMaxConnectionOnRPS Maximum connection on 5000 rps
	DefaultMaxConnectionOnRPS = 5000
)

// RabbitServer Server configuration
type RabbitServer struct {
	// RabbitMQx virtual host
	Vhost string
	// RabbitMQ host
	Host string
	// RabbitMQ port
	Port int
	// RabbitMQ user
	User string
	// RabbitMQ password
	Password string
	// Maximum number of connections to server
	MaxConnections int `yaml:"maxPublishConnections"`
	// Maximum lifetime for idle connection
	MaxIdleConnectionLifeTime time.Duration `yaml:"maxIdleConnectionLifeTime"`
}

// Get connection string
func (srv *RabbitServer) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%v/%s", srv.User, srv.Password, srv.Host, srv.Port, srv.Vhost)
}

// init default parameters
func (srv *RabbitServer) init() {
	if srv.MaxIdleConnectionLifeTime == 0 {
		srv.MaxIdleConnectionLifeTime = DefaultMaxIdleConnectionLifeTime
	}
	if srv.MaxConnections == 0 {
		srv.MaxConnections = DefaultMaxConnections
	}
}
