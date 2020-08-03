package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/porterr"
	"github.com/streadway/amqp"
)

// Publisher
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
	cp := a.sp.GetConnectionPoolOrCreate(server)
	// Publish message
	e = cp.Publish(p, *srv, *q,  route...)
	if e == nil {
		a.GetLogger(gocli.LogLevelDebug).Infoln("SUCCESS PUBLISH: ", string(p.Body))
	} else {
		a.GetLogger(gocli.LogLevelDebug).Errorln("PUBLISH ERROR: ", e.Error())
	}
	return e
}