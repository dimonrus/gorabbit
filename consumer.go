package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gohelp"
	"github.com/dimonrus/porterr"
	"github.com/streadway/amqp"
	"runtime/debug"
	"time"
)

// Consumer entity
type Consumer struct {
	// Queue name
	Queue string
	// Server name
	Server string
	// Delivery process callback
	Callback func(d amqp.Delivery)
	// Subscribers count
	Count uint8
	// Stop all consumers
	stop chan struct{}
	// Subscribers
	subscribers []*subscriber
	// amqp Connection
	connection *amqp.Connection
	// amqp Channel
	channel *amqp.Channel
	// amqp Queue
	queue *amqp.Queue
}

// Internal subscriber struct
type subscriber struct {
	// Subscriber name
	name string
	// chan for stop subscribing
	stop chan struct{}
}

// Stop all subscribers
func (c *Consumer) Stop() {
	for i := range c.subscribers {
		c.subscribers[i].stop <- struct{}{}
	}
	c.subscribers = make([]*subscriber, 0)
	c.stop <- struct{}{}
}

// Check for subscribers
func (c *Consumer) HasSubscribers() bool {
	if len(c.subscribers) > 0 {
		return true
	}
	return false
}

// Get s subscribers
func (c *Consumer) SubscribersCount() uint8 {
	return uint8(len(c.subscribers))
}

// New subscribers
func (c *Consumer) NewSubscriber(name string) *subscriber {
	return &subscriber{
		name: name,
		stop: make(chan struct{}),
	}
}

// Subscribe for queue
func (c *Consumer) Subscribe(logger gocli.Logger) porterr.IError {
	for num := uint8(0); num < c.Count; num++ {
		logger.Infof(`Subscribe '%s' queue on server '%s'`, c.Queue, c.Server)
		// If consumer not created
		if c == nil || c.queue == nil || c.connection == nil || c.channel == nil {
			return porterr.NewF(porterr.PortErrorParam, "Init consumer first")
		}
		// Subscriber name
		name := fmt.Sprintf("Subscriber: %s-%s", c.queue.Name, gohelp.RandString(5))
		// Consume messages
		messages, err := c.channel.Consume(c.queue.Name, name, false, false, false, false, nil)
		if err != nil {
			return porterr.NewF(porterr.PortErrorParam, "Consume '%s' error: %s", c.Queue, err.Error())
		}
		s := c.NewSubscriber(name)
		c.subscribers = append(c.subscribers, s)
		// Listen queue messages
		go func() {
			for {
				select {
				case d := <-messages:
					if d.Acknowledger == nil {
						break
					}
					logger.Infof("%s - received a message: \n %s", name, d.Body)
					func() {
						defer func() {
							if r := recover(); r != nil {
								logger.Errorf("%s - recovered in error: \n %s \n %s", name, r, debug.Stack())
								// Reject and requeue after 10 second pause
								time.Sleep(time.Second * 10)
								err := d.Reject(true)
								if err != nil {
									logger.Errorf("Reject message error: %s\n", err.Error())
								}
							}
						}()
						c.Callback(d)
						err := d.Ack(false)
						if err != nil {
							logger.Errorf("Ack message error: %s\n", err.Error())
							return
						}
					}()
				case <-s.stop:
					logger.Warnf("Stop: %v \n", name)
					return
				}
			}
		}()
	}
	return nil
}
