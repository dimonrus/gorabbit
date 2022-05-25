package test

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gorabbit"
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

type rConfig struct {
	Arguments gocli.Arguments
	Rabbit    gorabbit.Config
}

var cfg rConfig
var registry = map[string]*gorabbit.Consumer{
	"test": {Queue: "golkp.test", Server: "local", Callback: tTestConsume, Count: 5},
	"fan":  {Queue: "golkp.fanout", Server: "local", Callback: tTestConsumeFanout, Count: 2},
}

func tTestConsume(d amqp.Delivery) {
	fmt.Println("Тестовое сообщение успешно получено: " + string(d.Body))
}

var fanoutProcessed int32

func tTestConsumeFanout(d amqp.Delivery) {
	atomic.AddInt32(&fanoutProcessed, 1)
	fmt.Println("Тестовое сообщение успешно получено: "+string(d.Body), " - ", fanoutProcessed)
}

// Init test application
func testInitApp() *gorabbit.Application {
	rootPath, err := filepath.Abs("")
	if err != nil {
		panic(err)
	}

	app := gocli.NewApplication("global", rootPath+"/config", &cfg)
	app.ParseFlags(&cfg.Arguments)

	a := gorabbit.NewApplication(cfg.Rabbit, app).SetRegistry(registry)

	a.AttentionMessage("Starting AMQP Application...")

	return a
}

func TestApplication_Consume(t *testing.T) {
	a := testInitApp()
	go func() {
		e := a.Start(":3333", a.ConsumerCommander)
		if e != nil {
			a.GetLogger().Error(e)
		}
	}()

	command := gocli.ParseCommand([]byte("consumer start all"))
	a.ConsumerCommander(command)

	// Wait for OS signal
	forever := make(chan os.Signal, 1)

	a.GetLogger().Info(" [*] Waiting for messages. To exit press CTRL+C")
	signal.Notify(forever, os.Interrupt)
	<-forever

	a.GetLogger().Info(" [*] All Consumers is shutting down")
	os.Exit(0)
}

func TestApplication_Publish(t *testing.T) {
	app := testInitApp()
	pub := amqp.Publishing{
		Body: []byte("Hello my friend"),
	}
	commonLimit := 10_000
	rateLimit := 15
	limit := make(chan struct{}, rateLimit)
	for j := 0; j < commonLimit+rateLimit; j++ {
		limit <- struct{}{}
		if j >= commonLimit {
			continue
		}
		go func(i int, pub amqp.Publishing) {
			defer func() { <-limit }()
			pub.Body = []byte("hello:" + strconv.Itoa(i))
			e := app.Publish(pub, "golkp.test", "local")
			if e != nil {
				app.GetLogger().Errorln(e)
			}
		}(j, pub)
	}
	close(limit)
	app.GetLogger().Infoln("End publish!!!")
	c := make(chan int)
	<-c
}

func TestApplication_PublishFanout(t *testing.T) {
	app := testInitApp()
	pub := amqp.Publishing{
		Body: []byte("Hello my friend"),
	}
	for j := 0; j < 10; j++ {
		pub.Body = []byte("hello 1:" + strconv.Itoa(j))
		go app.Publish(pub, "golkp.fanout", "local", "#")
		time.Sleep(time.Millisecond * 1)
	}
	app.GetLogger().Infoln("End publish!!!")
	c := make(chan int)
	<-c
}
