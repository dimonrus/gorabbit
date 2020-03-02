package test

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gorabbit"
	"github.com/streadway/amqp"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"
)

type rConfig struct {
	Arguments gocli.Arguments
	Rabbit    gorabbit.Config
}

var cfg rConfig
var registry = map[string]*gorabbit.Consumer{
	"test": {Queue: "golkp.test", Server: "local", Callback: tTestConsume, Count: 2},
	"report": {Queue: "golkp.report", Server: "local", Callback: tTestConsume, Count: 4},
}

func tTestConsume(d amqp.Delivery) {
	fmt.Println("Тестовое сообщение успешно получено: " + string(d.Body))
}

func TestApplication_Consume(t *testing.T) {
	rootPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}

	app := gocli.NewApplication("global", rootPath+"/config", &cfg)
	app.ParseFlags(&cfg.Arguments)

	a := gorabbit.NewApplication(cfg.Rabbit, registry, app)

	a.GetLogger(gocli.LogLevelDebug).Info("Starting AMQP Application...")
	go func() {
		e := a.Start("3333", a.ConsumerCommander)
		if e != nil {
			a.GetLogger(gocli.LogLevelErr).Error(e)
		}
	}()

	go func() {
		command := gocli.ParseCommand([]byte("consumer start all"))
		a.ConsumerCommander(command)
	}()
	time.Sleep(time.Second * 1)
	go func() {
		return
		pub := amqp.Publishing{
			Body: []byte("Hello my friend"),
		}
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
		go a.Publish(pub, "golkp.test", "local")
	}()

	// Wait for OS signal
	forever := make(chan os.Signal, 1)

	a.GetLogger(gocli.LogLevelDebug).Info(" [*] Waiting for messages. To exit press CTRL+C")
	signal.Notify(forever, os.Interrupt)
	<-forever

	a.GetLogger(gocli.LogLevelDebug).Info(" [*] All Consumers is shutting down")
	os.Exit(0)
}
