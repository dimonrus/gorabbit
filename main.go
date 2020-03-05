package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gohelp"
	"github.com/dimonrus/porterr"
	"github.com/streadway/amqp"
	"time"
)

// Get server
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

// Get queue
func (c *Config) GetQueue(name string) (*RabbitQueue, porterr.IError) {
	queue, ok := c.Queues[name]
	if !ok {
		return nil, porterr.NewF(porterr.PortErrorSystem, "queue %s not found in rabbit config", name)
	}
	queue.Name = name
	return &queue, nil
}

// Get connection string
func (srv *RabbitServer) String() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%v/%s", srv.User, srv.Password, srv.Host, srv.Port, srv.Vhost)
}

// New rabbit Application
func NewApplication(config Config, app gocli.Application) *Application {
	return &Application{
		config:      config,
		Application: app,
	}
}

// Set Registry
func (a *Application) SetRegistry(r Registry) *Application{
	a.registry = r
	return a
}

// Get Config
func (a *Application) GetConfig() *Config {
	return &a.config
}

// Success message
func (a *Application) SuccessMessage(message string, command gocli.Command) {
	message = gohelp.AnsiGreen + message + gohelp.AnsiReset
	a.GetLogger(gocli.LogLevelInfo).Infoln(message)
	e := command.Result([]byte(message + "\n"))
	if e != nil {
		a.GetLogger(gocli.LogLevelWarn).Errorln(e)
	}
	return
}

// Success message
func (a *Application) AttentionMessage(message string, command gocli.Command) {
	message = gohelp.AnsiCyan + message + gohelp.AnsiReset
	a.GetLogger(gocli.LogLevelWarn).Warnln(message)
	e := command.Result([]byte(message + "\n"))
	if e != nil {
		a.GetLogger(gocli.LogLevelWarn).Errorln(e)
	}
	return
}

// Success message
func (a *Application) FailMessage(message string, command gocli.Command) {
	message = gohelp.AnsiRed + message + gohelp.AnsiReset
	a.GetLogger(gocli.LogLevelErr).Errorln(message)
	e := command.Result([]byte(message + "\n"))
	if e != nil {
		a.GetLogger(gocli.LogLevelWarn).Errorln(e)
	}
	return
}

// Get Registry
func (a *Application) GetRegistry() *Registry {
	return &a.registry
}

// Create new consumer
func (a *Application) Consume(name string) porterr.IError {
	consumer, ok := a.registry[name]
	if !ok {
		return porterr.NewF(porterr.PortErrorParam, "Consumer '%s' not found in registry")
	}
	// Get server
	srv, e := a.config.GetServer(consumer.Server)
	if e != nil {
		return e
	}
	// Get Queue
	q, e := a.config.GetQueue(consumer.Queue)
	if e != nil {
		return e
	}
	if q.Exchange == "" {
		e = porterr.New(porterr.PortErrorParam, "exchange is not defined")
		return e
	}
	consumer.stop = make(chan bool)
	var err error
	// Dial to server
	consumer.connection, err = amqp.Dial(srv.String())
	if err != nil {
		e = porterr.NewF(porterr.PortErrorConnection, "Failed connect to %s RabbitMQ Server", srv.Host)
		return e
	}
	// Get channel
	consumer.channel, err = consumer.connection.Channel()
	if err != nil {
		e = porterr.NewF(porterr.PortErrorConnection, "RabbitMQ Channel Error")
		return e
	}
	// Close channel and connection on return
	defer func() {
		// Close channel
		err := consumer.channel.Close()
		if err != nil {
			a.FailMessage("Channel close error: "+err.Error(), gocli.Command{})
		}
		// Close connection
		err = consumer.connection.Close()
		if err != nil {
			a.FailMessage("Connection close error: "+err.Error(), gocli.Command{})
		}
	}()
	// Init exchange
	err = consumer.channel.ExchangeDeclare(q.Exchange, q.Type, q.Durable, q.AutoDelete, q.Internal, q.Nowait, q.Arguments)
	if err != nil {
		e = porterr.NewF(porterr.PortErrorConnection, "Failed to declare exchange: '%s'", q.Name)
		return e
	}
	// Declare queue
	consumer.queue = new(amqp.Queue)
	*consumer.queue, err = consumer.channel.QueueDeclare(q.Name, q.Durable, q.AutoDelete, q.Exclusive, q.Nowait, q.Arguments)
	if err != nil {
		e = porterr.NewF(porterr.PortErrorConnection, "Failed to declare a queue: '%s'", q.Name)
		return e
	}
	// Walk on routing keys
	for _, key := range q.RoutingKey {
		// Bind queue for routing key
		err = consumer.channel.QueueBind(consumer.queue.Name, key, q.Exchange, q.Nowait, q.Arguments)
		if err != nil {
			e = porterr.NewF(porterr.PortErrorConnection, "Failed to bind a queue: '%s' for key '%s'", q.Name, key)
			return e
		}
	}
	ce := make(chan *amqp.Error)
	// Listen unexpected close the channel
	go func() {
		consumer.channel.NotifyClose(ce)
		select {
		case ae := <-ce:
			if ae != nil {
				a.FailMessage("Channel closed: "+ae.Error(), gocli.Command{})
				// Exit from child goroutine
				e = porterr.New(porterr.PortErrorSystem, ae.Error())
				consumer.Stop()
			}
		}
	}()
	e = consumer.Subscribe(a.GetLogger(gocli.LogLevelDebug))
	if e != nil {
		return e
	}
	a.SuccessMessage(fmt.Sprintf("Subscribers for '%s' are started", name), gocli.Command{})
	// Wait until consumer stop
	<-consumer.stop
	a.SuccessMessage("Close consuming for queue: "+consumer.Queue, gocli.Command{})
	return e
}

// Consumer command processor
func (a *Application) ConsumerCommander(command gocli.Command) {
	a.SuccessMessage("Receive command: "+command.String(), gocli.Command{})
	action, args, e := ParseCommand(command)
	if e != nil {
		a.FatalError(e)
		return
	}
	switch action {
	case CommandStart:
		for name := range *a.GetRegistry() {
			if args[0].GetString() == CommandKeyWordAll {
				if (*a.GetRegistry())[name].HasSubscribers() {
					a.AttentionMessage(fmt.Sprintf("Subscribers for '%s' already started", name), command)
					continue
				}
				a.SuccessMessage(fmt.Sprintf("Starting subscribe for '%s' consumer", name), command)
				go func(n string) {
					e := a.Consume(n)
					if e != nil {
						a.FailMessage(e.Error(), command)
						time.Sleep(time.Second)
						a.ConsumerCommander(command)
					}
				}(name)
			} else {
				for _, v := range args {
					if v.GetString() == name {
						if (*a.GetRegistry())[name].HasSubscribers() {
							a.AttentionMessage(fmt.Sprintf("Subscribers for '%s' already started", name), command)
							continue
						}
						a.SuccessMessage(fmt.Sprintf("Starting subscribe for '%s' consumer", name), command)
						go func(n string) {
							e := a.Consume(n)
							if e != nil {
								a.FailMessage(e.Error(), command)
								time.Sleep(time.Second)
								a.ConsumerCommander(command)
							}
						}(name)
					}
				}
			}
		}
	case CommandStop:
		for name := range *a.GetRegistry() {
			if args[0].GetString() == CommandKeyWordAll {
				if !(*a.GetRegistry())[name].HasSubscribers() {
					a.AttentionMessage(fmt.Sprintf("Subscribers for '%s' already stopped", name), command)
					continue
				}
				a.AttentionMessage(fmt.Sprintf("Stopping subscribers for '%s'", name), command)
				(*a.GetRegistry())[name].Stop()
			} else {
				for _, v := range args {
					if v.GetString() == name {
						if !(*a.GetRegistry())[name].HasSubscribers() {
							a.AttentionMessage(fmt.Sprintf("Subscribers for '%s' already stopped", name), command)
							continue
						}
						a.AttentionMessage(fmt.Sprintf("Stopping subscribers for '%s'", name), command)
						(*a.GetRegistry())[name].Stop()
					}
				}
			}
		}
	case CommandRestart:
		for name := range *a.GetRegistry() {
			if args[0].GetString() == CommandKeyWordAll {
				if (*a.GetRegistry())[name].HasSubscribers() {
					a.AttentionMessage(fmt.Sprintf("Stopping subscribers for '%s'", name), command)
					(*a.GetRegistry())[name].Stop()
				}
				a.SuccessMessage(fmt.Sprintf("Starting subscribe for '%s' consumer", name), command)
				go func(n string) {
					e := a.Consume(n)
					if e != nil {
						a.FailMessage(e.Error(), command)
						time.Sleep(time.Second)
						a.ConsumerCommander(command)
					}
				}(name)
			} else {
				for _, v := range args {
					if v.GetString() == name {
						if (*a.GetRegistry())[name].HasSubscribers() {
							a.AttentionMessage(fmt.Sprintf("Stopping subscribers for '%s'", name), command)
							(*a.GetRegistry())[name].Stop()
						}
						a.SuccessMessage(fmt.Sprintf("Starting subscribe for '%s' consumer", name), command)
						go func(n string) {
							e := a.Consume(n)
							if e != nil {
								a.FailMessage(e.Error(), command)
								time.Sleep(time.Second)
								a.ConsumerCommander(command)
							}
						}(name)
					}
				}
			}
		}
	case CommandStatus:
		for name := range *a.GetRegistry() {
			if args[0].GetString() == CommandKeyWordAll {
				a.SuccessMessage(fmt.Sprintf("Consumer '%s' have a %v subscribers", name, (*a.GetRegistry())[name].SubscribersCount()), command)
			} else {
				for _, v := range args {
					if v.GetString() == name {
						a.SuccessMessage(fmt.Sprintf("Consumer '%s' have a %v subscribers", name, (*a.GetRegistry())[name].SubscribersCount()), command)
					}
				}
			}
		}
	case CommandSet:
		if args[0].GetString() == CommandKeyWordCount {
			count := args[1].GetInt()
			for name := range *a.GetRegistry() {
				for _, v := range args[2:] {
					if v.GetString() == name {
						a.SuccessMessage(fmt.Sprintf("Consumer '%s' set subscribers count to: %v ", name, count), command)
						(*a.GetRegistry())[name].Count = uint8(count)
					}
				}
			}
		} else {
			a.AttentionMessage(fmt.Sprintf("Unknown set command: "+command.GetOrigin()), command)
		}
	default:
		a.AttentionMessage(fmt.Sprintf("Unknown command: "+command.GetOrigin()), command)
	}
}
