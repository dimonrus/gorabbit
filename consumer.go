package gorabbit

import (
	"errors"
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gohelp"
	"github.com/streadway/amqp"
	"os"
	"os/signal"
	"runtime/debug"
	"time"
)

// On Error
func (c *Consumer) onError(err error, msg string) {
	if err != nil {
		c.app.GetLogger(gocli.LogLevelDebug).Errorf("%s: %s", msg, err)
		c.app.FatalError(fmt.Errorf("%s: %s", msg, err))
	}
}

// Notify closing channel
func (c *Consumer) notifyChannelClose() {
	ce := make(chan *amqp.Error)
	c.channel.NotifyClose(ce)
	for {
		select {
		case cce := <-ce:
			if cce != nil {
				c.app.GetLogger(gocli.LogLevelDebug).Error("Channel closed: ", cce, "queue:", c.queue.Name)
			}
		}
	}
}

// Create new consumer
func NewConsumer(app gocli.Application, s RabbitServer, q RabbitQueue, callback func(d amqp.Delivery)) *Consumer {
	var err error
	c := Consumer{app: app}

	connectionString := fmt.Sprintf("amqp://%s:%s@%s:%v/%s", s.User, s.Password, s.Host, s.Port, s.Vhost)

	c.connection, err = amqp.Dial(connectionString)
	c.onError(err, fmt.Sprintf("Failed connect to %s RabbitMQ Server", s.Host))

	c.channel, err = c.connection.Channel()
	c.onError(err, fmt.Sprintf("RabbitMQ Channel Error"))

	if q.Exchange == "" {
		err = errors.New("exchange is not defined")
		c.onError(err, err.Error())
	}

	err = c.channel.ExchangeDeclare(q.Exchange, q.Type, q.Durable, q.AutoDelete, q.Internal, q.Nowait, q.Arguments)
	c.onError(err, fmt.Sprintf("Failed to declare exchange: %s", q.Name))

	queue, err := c.channel.QueueDeclare(q.Name, q.Durable, q.AutoDelete, q.Exclusive, q.Nowait, q.Arguments)
	c.onError(err, fmt.Sprintf("Failed to declare a queue: %s", q.Name))

	c.queue = &queue
	c.process = callback

	if q.Exchange != "" {
		for _, key := range q.RoutingKey {
			err = c.channel.QueueBind(queue.Name, key, q.Exchange, q.Nowait, q.Arguments)
			c.onError(err, fmt.Sprintf("Failed to declare a queue: %s", q.Name))
		}
	}

	return &c
}

// Subscribe
func (c *Consumer) Subscribe() {
	rndStr := gohelp.RandString(5)
	name := fmt.Sprintf("Consumer: %s-%s", c.queue.Name, rndStr)

	messages, err := c.channel.Consume(c.queue.Name, name, false, false, false, false, nil)
	c.onError(err, fmt.Sprintf("Failed to register a consumer for queue: %s", c.queue.Name))

	defer c.connection.Close()
	defer c.channel.Close()
	go c.notifyChannelClose()

	exit := make(chan bool)

	go func() {
		for d := range messages {
			c.app.GetLogger(gocli.LogLevelDebug).Infof("%s - received a message: \n %s", name, d.Body)
			func() {
				defer func() {
					if r := recover(); r != nil {
						c.app.GetLogger(gocli.LogLevelDebug).Errorf("%s - recovered in error: \n %s \n %s", name, r, debug.Stack())
						time.Sleep(time.Second * 10)
						err := d.Reject(true)
						if err != nil {
							c.app.GetLogger(gocli.LogLevelDebug).Errorf("Reject message error: %s", err.Error())
						}
					}
				}()
				c.process(d)
				err := d.Ack(true)
				if err != nil {
					c.app.GetLogger(gocli.LogLevelDebug).Errorf("Ack message error: %s", err.Error())
				}
			}()
		}
	}()

	<-exit
}

// Consume single
func consumeSingle(app gocli.Application, server string, queue string, cfg Config, callback func(d amqp.Delivery)) {
	// Get server
	rs, oks := cfg.Servers[server]
	if !oks {
		app.GetLogger(gocli.LogLevelDebug).Warn("%s server not found in config", server)
		return
	}
	// Get queue
	rq, okq := cfg.Queues[queue]
	if !okq {
		app.GetLogger(gocli.LogLevelDebug).Warn("%s queue not found in config", queue)
		return
	}
	// Assign name
	rq.Name = queue
	app.GetLogger(gocli.LogLevelDebug).Infof(`Starting consumer on server: "%s" for queue "%s"`, server, queue)
	// Subscribe
	consumer := NewConsumer(app, rs, rq, callback)
	if consumer != nil {
		go consumer.Subscribe()
	}
}

// Start consumer application
func Start(app gocli.Application, registry Registry, cfg Config, arguments gocli.Arguments) {
	app.GetLogger(gocli.LogLevelDebug).Info("Starting AMQP Application...")
	consumerName := arguments["name"].GetString()

	forever := make(chan os.Signal, 1)

	for registryName, registryItem := range registry {
		if len(consumerName) == 0 {
			for num := byte(0); num < registryItem.Count; {
				consumeSingle(app, registryItem.Server, registryItem.Queue, cfg, registryItem.Callback)
				num++
			}
		} else {
			if consumerName == registryName {
				for num := byte(0); num < registryItem.Count; {
					consumeSingle(app, registryItem.Server, registryItem.Queue, cfg, registryItem.Callback)
					num++
				}
			}
		}
	}

	app.GetLogger(gocli.LogLevelDebug).Info(" [*] Waiting for messages. To exit press CTRL+C")
	signal.Notify(forever, os.Interrupt)
	<-forever

	app.GetLogger(gocli.LogLevelDebug).Info(" [*] All Consumers is shutting down")
	os.Exit(0)
}
