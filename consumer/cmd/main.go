package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/anton-kapralov/currency-market-pulse/consumer/consuming"
	"github.com/anton-kapralov/currency-market-pulse/consumer/http/rest"
	"github.com/anton-kapralov/currency-market-pulse/consumer/ratelimit"
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

func newRedisClient(host string, port int) *redis.Client {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Connecting to Redis at %s", addr)
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	statusCmd := client.Ping(context.Background())
	if statusCmd.Err() != nil {
		log.Fatalln(statusCmd.Err())
	}
	return client
}

type options struct {
	kafka struct {
		host string
		port int
	}
	redis struct {
		host string
		port int
	}
	rateLimiter struct {
		duration time.Duration
		limit    int
	}
}

func main() {
	var opts options
	flag.StringVar(&opts.kafka.host, "kafka.host", "localhost", "Kafka host")
	flag.IntVar(&opts.kafka.port, "kafka.port", 9092, "Kafka port")
	flag.StringVar(&opts.redis.host, "redis.host", "localhost", "Redis host")
	flag.IntVar(&opts.redis.port, "redis.port", 6379, "Redis port")
	flag.DurationVar(&opts.rateLimiter.duration, "ratelimit.duration", 0, "Limit request per this time unit")
	flag.IntVar(&opts.rateLimiter.limit, "ratelimit.limit", 0, "Limit request per this time unit")
	flag.Parse()

	kafkaProducer := newKafkaProducer(opts.kafka.host, opts.kafka.port)
	consumer := consuming.NewService(kafkaProducer, "currency-trades")
	redisClient := newRedisClient(opts.redis.host, opts.redis.port)
	rateLimiter := ratelimit.NewRateLimiter(redisClient, opts.rateLimiter.duration, opts.rateLimiter.limit)
	restController := rest.NewController(consumer, rateLimiter)

	router := gin.Default()
	router.POST("/api/trade", restController.SaveTradeMessage)

	log.Fatalln(router.Run(":8081"))
}
