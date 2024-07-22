package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/IBM/sarama"

	"github.com/anton-kapralov/currency-market-pulse/importer/importing"
)

func newKafkaConsumerGroup(host string, port int) sarama.ConsumerGroup {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Connecting to Kafka at %s", addr)
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	kafkaConsumerGroup, err := sarama.NewConsumerGroup([]string{addr}, "importer", config)
	if err != nil {
		log.Fatalf("Error creating consumer group client: %v", err)
	}
	return kafkaConsumerGroup
}

func newClickhouseClient(host string, port int) *ch.Client {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Connecting to Clickhouse at %s", addr)
	ctx := context.Background()
	c, err := ch.Dial(ctx, ch.Options{
		Address:  addr,
		Database: "cmp",
	})
	if err != nil {
		log.Fatalf("Failed to connect to Clickhouse DB: %s", err)
	}
	if err := c.Ping(ctx); err != nil {
		var exception *clickhouse.Exception
		if errors.As(err, &exception) {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		log.Fatalf("Failed to ping Clickhouse DB: %s", err)
	}
	return c
}

func onInterrupt(cancel context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		for s := range ch {
			os.Stderr.WriteString(fmt.Sprintln(s.String()))
			cancel()
		}
	}()
}

type options struct {
	kafka struct {
		host string
		port int
	}
	clickhouse struct {
		host string
		port int
	}
}

func main() {
	var opts options
	flag.StringVar(&opts.kafka.host, "kafka.host", "localhost", "Kafka host")
	flag.IntVar(&opts.kafka.port, "kafka.port", 9092, "Kafka port")
	flag.StringVar(&opts.clickhouse.host, "clickhouse.host", "localhost", "Kafka host")
	flag.IntVar(&opts.clickhouse.port, "clickhouse.port", 9000, "Kafka port")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	onInterrupt(cancel)

	kafkaConsumerGroup := newKafkaConsumerGroup(opts.kafka.host, opts.kafka.port)
	clickhouseClient := newClickhouseClient(opts.clickhouse.host, opts.clickhouse.port)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	service := importing.NewService(kafkaConsumerGroup, "currency-trades", clickhouseClient)
	service.Start(ctx, wg)

	log.Println("Ready")
	wg.Wait()
}
