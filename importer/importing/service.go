package importing

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/anton-kapralov/currency-market-pulse/proto-gen/currencymarketpb"
	"google.golang.org/protobuf/proto"
	"log"
	"sync"
)

type Service interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
}

type service struct {
	kafkaConsumerGroup sarama.ConsumerGroup
	topics             []string
}

func NewService(kafkaConsumerGroup sarama.ConsumerGroup, topic string) Service {
	return &service{
		kafkaConsumerGroup: kafkaConsumerGroup,
		topics:             []string{topic},
	}
}

func (s *service) Start(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := s.kafkaConsumerGroup.Consume(ctx, s.topics, s); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
		}
	}()
}

func (s *service) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (s *service) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (s *service) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Printf("message channel was closed")
				return nil
			}
			trade := &currencymarketpb.Trade{}
			if err := proto.Unmarshal(message.Value, trade); err != nil {
				return fmt.Errorf("failed to unmarshal trade message: %s", err)
			}
			log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", trade, message.Timestamp, message.Topic)
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}
