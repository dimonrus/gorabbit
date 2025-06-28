package gorabbit

import (
	"github.com/dimonrus/porterr"
)

// RabbitQueue Queue configuration
type RabbitQueue struct {
	// Name of server
	Server string
	// Name of exchange
	Exchange string
	// Do not accept publishing
	Internal bool
	// The common types are "direct", "fanout", "topic" and "headers".
	Type string
	// Name for queue
	Name string
	// DEPRECATED Always false
	Passive bool
	// Durable and Non-Auto-Deleted exchanges will survive server restarts and remain
	// declared when there are no remaining bindings.
	Durable bool
	// Exclusive queues are only accessible by the connection that declares them and
	// will be deleted when the connection closes.
	Exclusive bool
	// When noWait is true, the queue will assume it to be declared on the server.
	Nowait bool
	// Durable and Auto-Deleted queues will be restored on server restart, but without
	// active consumers will not survive and be removed.
	AutoDelete bool `yaml:"autoDelete"`
	// Prefetch settings
	Prefetch Prefetch
	// List of routing keys for queue
	RoutingKey []string `yaml:"routingKey"`
	// Queue custom arguments
	Arguments map[string]interface{}
}

// Registry consumer registry
type Registry map[string]*Consumer

// Servers server registry
type Servers map[string]RabbitServer

// Queues queue registry
type Queues map[string]RabbitQueue

// Config Rabbit config
type Config struct {
	// Server configurations
	Servers Servers
	// Queues configuration
	Queues Queues
}

// Prefetch options
type Prefetch struct {
	// count of prefetch items
	Count int
	// Size of prefetch
	Size int
}

// IsEmpty is prefetch empty
func (p Prefetch) IsEmpty() bool {
	return p.Count == 0 && p.Size == 0
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
