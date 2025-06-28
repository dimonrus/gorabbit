package test

import (
	"context"
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
	"test": {Queue: "rmq.test", Server: "local", Callback: tTestConsume, Count: 5},
	"fan1": {Queue: "rmq.fanout1", Server: "local", Callback: tTestConsumeFanout, Count: 1},
	"fan2": {Queue: "rmq.fanout2", Server: "local", Callback: tTestConsumeFanout, Count: 1},
}

func tTestConsume(d amqp.Delivery) {
	atomic.AddInt32(&fanoutProcessed, 1)
	//fmt.Println("Тестовое сообщение успешно получено: " + string(d.Body))
}

var fanoutProcessed int32

func idle(ctx context.Context, t *testing.T) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.Log(atomic.LoadInt32(&fanoutProcessed))
		case <-ctx.Done():
			return
		}
	}
}

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

	ctx := context.Background()
	go idle(ctx, t)

	a.GetLogger().Info(" [*] Waiting for messages. To exit press CTRL+C")
	signal.Notify(forever, os.Interrupt)
	<-forever
	ctx.Done()

	a.GetLogger().Info(" [*] All Consumers is shutting down")
	os.Exit(0)
}

func TestApplication_Publish(t *testing.T) {
	app := testInitApp()
	pub := amqp.Publishing{
		Body: []byte("Hello my friend"),
	}
	commonLimit := 1_000_000
	rateLimit := 15
	value := make(chan int)
	for i := 0; i < rateLimit; i++ {
		go func(i chan int, pub amqp.Publishing) {
			for v := range i {
				pub.Body = []byte("hello:" + strconv.Itoa(v))
				e := app.Publish(pub, "golkp.test", "local")
				if e != nil {
					t.Fatal(e)
				}
			}
		}(value, pub)
	}
	for j := 0; j < commonLimit; j++ {
		value <- j
	}
	app.GetLogger().Infoln("End publish!!!")
	c := make(chan int)
	<-c
}

func TestApplication_PublishFanout(t *testing.T) {
	app := testInitApp()
	pub := amqp.Publishing{
		Body: []byte("Hello my friend"),
	}
	for j := 0; j < 100000; j++ {
		pub.Body = []byte("hello 1:" + strconv.Itoa(j))
		e := app.Publish(pub, "rmq.fanout", "local", "#")
		if e != nil {
			t.Fatal(e)
		}
	}
	app.GetLogger().Infoln("End publish!!!")
	time.Sleep(time.Second * 10)
}
