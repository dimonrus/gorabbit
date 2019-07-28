package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/streadway/amqp"
	"net/http"
)

// Publisher
func (a *Application) Publish(publishing amqp.Publishing, queue string, server string) gocli.IError {
	// Get server config
	srv, e := a.config.GetServer(server)
	if e != nil {
		return e
	}
	// Get queue config
	que, e := a.config.GetQueue(queue)
	if e != nil {
		return e
	}
	// get connection string
	connection, err := amqp.Dial(srv.String())
	if err != nil {
		return gocli.NewError(fmt.Sprintf("Can't dial to RabbitMq server (%s): %s", srv.String(), err.Error()), http.StatusInternalServerError)
	}
	defer connection.Close()
	// get channel
	channel, err := connection.Channel()
	if err != nil {
		return gocli.NewError(fmt.Sprintf("Can't get channel from server (%s): %s", server, err.Error()), http.StatusInternalServerError)
	}
	// Set confirm mode
	if err := channel.Confirm(false); err != nil {
		return gocli.NewError("Confirm mode set failed: "+err.Error(), http.StatusInternalServerError)
	}
	// Publish notification
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	defer a.confirmOne(confirms)
	// Publish to all routing keys
	if len(que.RoutingKey) > 0 {
		for _, value := range que.RoutingKey {
			if e = a.publish(channel, que.Exchange, value, false, false, publishing); e != nil {
				return e
			}
		}
	} else {
		if e = a.publish(channel, que.Exchange, "", false, false, publishing); e != nil {
			return e
		}
	}

	return nil
}

// Publish message
func (a *Application) publish(channel *amqp.Channel, exchange, key string, mandatory, immediate bool, publishing amqp.Publishing) gocli.IError {
	var e gocli.IError
	err := channel.Publish(exchange, key, mandatory, immediate, publishing)
	a.base.GetLogger(gocli.LogLevelDebug).Infoln("PUBLISH: ", string(publishing.Body))
	if err != nil {
		e = gocli.NewError(fmt.Sprintf("exchange publish: %s", err.Error()), http.StatusInternalServerError)
		a.base.GetLogger(gocli.LogLevelDebug).Errorf("exchange publish: %s", err)
	}
	return e
}

// Check confirm
func (a *Application) confirmOne(confirms <-chan amqp.Confirmation) {
	if confirmed := <-confirms; confirmed.Ack {
		a.base.GetLogger(gocli.LogLevelDebug).Infof("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
	} else {
		a.base.GetLogger(gocli.LogLevelDebug).Errorf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
	}
}
