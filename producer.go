package gorabbit

import (
	"github.com/dimonrus/gohelp"
	"github.com/dimonrus/porterr"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

// Publish Publisher
// publishing - AMQP Message
// queue - name of the queue defined in config
// server - name of the server defined in config
// route - routing keys
func (a *Application) Publish(p amqp.Publishing, queue string, server string, route ...string) porterr.IError {
	// Get server config
	srv, e := a.GetConfig().GetServer(server)
	if e != nil {
		return e
	}
	// Set default setting if not set in config
	srv.init()
	// Get queue config
	q, e := a.GetConfig().GetQueue(queue)
	if e != nil {
		return e
	}
	// Define routing keys
	if len(route) == 0 {
		route = q.RoutingKey
	}
	if len(route) == 0 {
		route = append(route, "")
	}
	cp := a.sp.GetConnectionPoolOrCreate(server, srv.MaxConnections)
	// Publish a message
	for e = cp.Publish(p, *srv, *q, route...); e != nil; {
		a.GetLogger().Errorln(gohelp.Red("PUBLISH ERROR: " + e.Error()))
		time.Sleep(time.Millisecond * 1000)
	}
	return e
}
