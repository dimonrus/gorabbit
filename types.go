package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/streadway/amqp"
)

// Consumer registry
type Registry map[string]*Consumer

// Servers
type Servers map[string]RabbitServer

// Queues
type Queues map[string]RabbitQueue

// Rabbit config
type Config struct {
	Servers Servers
	Queues  Queues
}

// Internal subscriber struct
type subscriber struct {
	// Subscriber name
	name string
	// chan for stop subscribing
	stop chan bool
}

// Consumer entity
type Consumer struct {
	// Queue name
	Queue      string
	// Server name
	Server     string
	// Delivery process callback
	Callback   func(d amqp.Delivery)
	// Subscribers count
	Count      uint8
	// Stop all consumers
	stop       chan bool
	// Subscribers
	subscribers []*subscriber
	// amqp Connection
	connection *amqp.Connection
	// amqp Channel
	channel    *amqp.Channel
	// amqp Queue
	queue      *amqp.Queue
}

// Server configuration
type RabbitServer struct {
	// rabbit mq virtual host
	Vhost    string
	// rabbit mq host
	Host     string
	// rabbit mq port
	Port     int
	// rabbit mq user
	User     string
	// rabbit mq password
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
	registry Registry
	gocli.Application
}
