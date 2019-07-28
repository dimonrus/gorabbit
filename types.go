package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/streadway/amqp"
)

// Consumer registry
type Registry map[string]RegistryItem

// Servers
type Servers map[string]RabbitServer

// Queues
type Queues map[string]RabbitQueue

// Rabbit config
type Config struct {
	Servers Servers
	Queues  Queues
}

// Consumer registry item
type RegistryItem struct {
	Queue    string
	Server   string
	Callback func(d amqp.Delivery)
	Count    byte
}

// Consumer entity
type Consumer struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queue      *amqp.Queue
	process    func(d amqp.Delivery)
}

// Server configuration
type RabbitServer struct {
	Vhost    string
	Host     string
	Port     int
	User     string
	Password string
}

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

// Rabbit Application
type Application struct {
	config   Config
	base     gocli.Application
	registry *Registry
}
