package gorabbit

import (
	"github.com/dimonrus/porterr"
)

// Queue configuration
type RabbitQueue struct {
	Server     string
	Exchange   string
	Internal   bool
	Type       string
	Name       string
	Passive    bool
	Durable    bool
	Exclusive  bool
	Nowait     bool
	AutoDelete bool     `yaml:"autoDelete"`
	RoutingKey []string `yaml:"routingKey"`
	Arguments  map[string]interface{}
}

// Registry consumer registry
type Registry map[string]*Consumer

// Servers server registry
type Servers map[string]RabbitServer

// Queues queue registry
type Queues map[string]RabbitQueue

// Config Rabbit config
type Config struct {
	// Servers configuration
	Servers Servers
	// Queues configuration
	Queues Queues
}

// GetServer Get server
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

// GetQueue Get queue
func (c *Config) GetQueue(name string) (*RabbitQueue, porterr.IError) {
	queue, ok := c.Queues[name]
	if !ok {
		return nil, porterr.NewF(porterr.PortErrorSystem, "queue %s not found in rabbit config", name)
	}
	queue.Name = name
	return &queue, nil
}
