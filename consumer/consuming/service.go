package consuming

import (
	"fmt"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/anton-kapralov/currency-market-pulse/proto-gen/currencymarketpb"
)

type Service interface {
	Save(trade *currencymarketpb.Trade) error
}

type service struct {
	kafkaProducer sarama.AsyncProducer
	topic         string
}

func NewService(kafkaProducer sarama.AsyncProducer, topic string) Service {
	return &service{
		kafkaProducer: kafkaProducer,
		topic:         topic,
	}
}

func (c *service) Save(trade *currencymarketpb.Trade) error {
	bytes, err := proto.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to encode trade message: %s", err)
	}

	c.kafkaProducer.Input() <- &sarama.ProducerMessage{Topic: c.topic, Value: sarama.ByteEncoder(bytes)}
	return nil
}
