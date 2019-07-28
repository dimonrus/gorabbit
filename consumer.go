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

// Notify closing channel
func (a *Application) notifyChannelClose(channel *amqp.Channel) {
	ce := make(chan *amqp.Error)
	channel.NotifyClose(ce)
	for {
		select {
		case cce := <-ce:
			if cce != nil {
				a.base.GetLogger(gocli.LogLevelDebug).Error("Channel closed: ", cce)
			}
		}
	}
}

// Create new consumer
func (a *Application) NewConsumer(serverName string, queueName string, callback func(d amqp.Delivery)) *Consumer {
	var err error
	c := Consumer{}

	// Get server
	srv, e := a.config.GetServer(serverName)
	a.onError(e, "Server config is incorrect")

	// Get Queue
	q, e := a.config.GetQueue(queueName)
	a.onError(e, "Queue config is incorrect")
	if q.Exchange == "" {
		err = errors.New("exchange is not defined")
		a.onError(err, err.Error())
	}

	// Dial to server
	c.connection, err = amqp.Dial(srv.String())
	a.onError(err, fmt.Sprintf("Failed connect to %s RabbitMQ Server", srv.Host))

	// Get channel
	c.channel, err = c.connection.Channel()
	a.onError(err, fmt.Sprintf("RabbitMQ Channel Error"))

	// Init exchange
	err = c.channel.ExchangeDeclare(q.Exchange, q.Type, q.Durable, q.AutoDelete, q.Internal, q.Nowait, q.Arguments)
	a.onError(err, fmt.Sprintf("Failed to declare exchange: %s", q.Name))

	// Declare queue
	queue, err := c.channel.QueueDeclare(q.Name, q.Durable, q.AutoDelete, q.Exclusive, q.Nowait, q.Arguments)
	a.onError(err, fmt.Sprintf("Failed to declare a queue: %s", q.Name))

	// Populate struct Queue
	c.queue = &queue
	c.process = callback

	// Walk on routing keys
	for _, key := range q.RoutingKey {
		// Bind queue for routing key
		err = c.channel.QueueBind(queue.Name, key, q.Exchange, q.Nowait, q.Arguments)
		a.onError(err, fmt.Sprintf("Failed to declare a queue: %s", q.Name))
	}

	return &c
}

// Subscribe
func (a *Application) Subscribe(consumer *Consumer) {
	// Consumer name
	rndStr := gohelp.RandString(5)
	name := fmt.Sprintf("Consumer: %s-%s", consumer.queue.Name, rndStr)

	// Consume messages
	messages, err := consumer.channel.Consume(consumer.queue.Name, name, false, false, false, false, nil)
	a.onError(err, fmt.Sprintf("Failed to register a consumer for queue: %s", consumer.queue.Name))

	// Close connection
	defer consumer.connection.Close()
	// Close channel
	defer consumer.channel.Close()
	// Notify close
	go a.notifyChannelClose(consumer.channel)

	exit := make(chan bool)

	go func() {
		for d := range messages {
			a.base.GetLogger(gocli.LogLevelDebug).Infof("%s - received a message: \n %s", name, d.Body)
			func() {
				defer func() {
					if r := recover(); r != nil {
						a.base.GetLogger(gocli.LogLevelDebug).Errorf("%s - recovered in error: \n %s \n %s", name, r, debug.Stack())
						time.Sleep(time.Second * 10)
						err := d.Reject(true)
						if err != nil {
							a.base.GetLogger(gocli.LogLevelDebug).Errorf("Reject message error: %s", err.Error())
						}
					}
				}()
				consumer.process(d)
				err := d.Ack(true)
				if err != nil {
					a.base.GetLogger(gocli.LogLevelDebug).Errorf("Ack message error: %s", err.Error())
				}
			}()
		}
	}()

	<-exit
}

// Consume single
func (a *Application) consumeSingle(item RegistryItem) {
	a.base.GetLogger(gocli.LogLevelDebug).Infof(`Starting consumer on server: "%s" for queue "%s"`, item.Server, item.Queue)
	// Subscribe
	consumer := a.NewConsumer(item.Server, item.Queue, item.Callback)
	if consumer != nil {
		go a.Subscribe(consumer)
	}
}

// Start consumer Application
func (a *Application) Consume(registry Registry, arguments gocli.Arguments) {
	a.base.GetLogger(gocli.LogLevelDebug).Info("Starting AMQP Application...")
	consumerName := arguments["name"].GetString()

	forever := make(chan os.Signal, 1)

	// Registry item iterator
	for registryName, registryItem := range registry {
		if len(consumerName) == 0 {
			for num := byte(0); num < registryItem.Count; {
				a.consumeSingle(registryItem)
				num++
			}
		} else {
			if consumerName == registryName {
				for num := byte(0); num < registryItem.Count; {
					a.consumeSingle(registryItem)
					num++
				}
			}
		}
	}

	a.base.GetLogger(gocli.LogLevelDebug).Info(" [*] Waiting for messages. To exit press CTRL+C")
	signal.Notify(forever, os.Interrupt)
	<-forever

	a.base.GetLogger(gocli.LogLevelDebug).Info(" [*] All Consumers is shutting down")
	os.Exit(0)
}
