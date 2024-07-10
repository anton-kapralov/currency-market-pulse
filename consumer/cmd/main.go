package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"

	"github.com/anton-kapralov/currency-market-pulse/consumer/consuming"
	"github.com/anton-kapralov/currency-market-pulse/consumer/http/rest"
)

func newKafkaProducer(host string, port int) sarama.AsyncProducer {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Connecting to Kafka at %s", addr)
	config := sarama.NewConfig()
	config.Producer.Idempotent = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	config.Net.MaxOpenRequests = 1
	kafkaProducer, err := sarama.NewAsyncProducer([]string{addr}, config)
	if err != nil {
		log.Fatalf("Failed to create a Kafka producer: %s", err)
	}
	return kafkaProducer
}

type options struct {
	kafka struct {
		host string
		port int
	}
}

func main() {
	var opts options
	flag.StringVar(&opts.kafka.host, "kafka.host", "localhost", "Kafka host")
	flag.IntVar(&opts.kafka.port, "kafka.port", 9092, "Kafka port")
	flag.Parse()

	kafkaProducer := newKafkaProducer(opts.kafka.host, opts.kafka.port)
	consumer := consuming.NewService(kafkaProducer, "currency-trades")
	restController := rest.NewController(consumer)

	router := gin.Default()
	router.POST("/api/trade", restController.SaveTradeMessage)

	log.Fatalln(router.Run(":8081"))
}
