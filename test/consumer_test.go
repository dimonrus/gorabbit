package test

import (
	"fmt"
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/gorabbit"
	"github.com/streadway/amqp"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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
	"report": {Queue: "golkp.report", Server: "local", Callback: tTestConsume, Count: 0},
}

func tTestConsume(d amqp.Delivery) {
	fmt.Println("Тестовое сообщение успешно получено: " + string(d.Body))
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
	for j := 0; j< 10000; j++ {
		pub.Body = []byte("hello 1:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 2:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 3:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 4:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 5:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 6:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 7:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 8:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 9:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		pub.Body = []byte("hello 10:"+strconv.Itoa(j))
		go app.Publish(pub, "golkp.test", "local")
		time.Sleep(time.Millisecond * 1)
	}
	c := make(chan int)
	<-c
}
