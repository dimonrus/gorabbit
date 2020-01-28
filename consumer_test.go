package gorabbit

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/streadway/amqp"
	"testing"
)
type rConfig struct {
	Arguments    gocli.Arguments
	Rabbit       Config
}
var cfg rConfig
var registry = make(Registry)

type application struct {
	gocli.Application
	rabbit         *Application
}

// logger
func (app application) GetLogger(l int) gocli.Logger {
	return app.Application.GetLogger(gocli.LogLevelDebug)
}

func tTestConsume(d amqp.Delivery) {
	fmt.Println("Тестовое сообщение успешно получено")
}

func TestApplication_Consume(t *testing.T) {
	registry["test"] = RegistryItem{Queue: "golkp.test", Server: "local", Callback: tTestConsume, Count: 1}
	app := gocli.DNApp{
		ConfigPath: "config",
	}.New("global", &cfg)
	a := application{
		Application: app,
	}
	s := ""
	name := gocli.Argument{
		Type:"string",
		Value:&s,
	}
	cfg.Arguments["name"] = name
	a.rabbit = NewApplication(cfg.Rabbit, a)
	a.rabbit.Consume(registry, cfg.Arguments)
}
