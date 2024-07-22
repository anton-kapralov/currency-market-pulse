package importing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/anton-kapralov/currency-market-pulse/proto-gen/currencymarketpb"
)

type Service interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
}

type service struct {
	kafka  sarama.ConsumerGroup
	topics []string
	db     *ch.Client
}

func NewService(kafka sarama.ConsumerGroup, topic string, db *ch.Client) Service {
	return &service{
		kafka:  kafka,
		topics: []string{topic},
		db:     db,
	}
}

func (s *service) Start(ctx context.Context, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := s.kafka.Consume(ctx, s.topics, s); err != nil {
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
			if err := s.saveTrade(trade); err != nil {
				log.Println(err)
				return err
			}
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

func (s *service) saveTrade(trade *currencymarketpb.Trade) error {
	var (
		userId             chproto.ColStr
		currencyFrom       = new(chproto.ColStr).LowCardinality()
		currencyTo         = new(chproto.ColStr).LowCardinality()
		amountSellMicros   chproto.ColUInt64
		amountBuyMicros    chproto.ColUInt64
		rate               chproto.ColFloat64
		originatingCountry = new(chproto.ColStr).LowCardinality()
		timePlaced         = new(chproto.ColDateTime64).WithPrecision(chproto.PrecisionMilli)
	)

	userId.Append(trade.UserId)
	currencyFrom.Append(trade.CurrencyFrom)
	currencyTo.Append(trade.CurrencyTo)
	amountSellMicros.Append(uint64(trade.AmountSellMicros))
	amountBuyMicros.Append(uint64(trade.AmountBuyMicros))
	rate.Append(float64(trade.AmountBuyMicros) / float64(trade.AmountSellMicros))
	originatingCountry.Append(trade.OriginatingCountry)
	timePlaced.Append(time.Unix(0, trade.TimePlacedMs*int64(time.Millisecond)))

	ctx := context.Background()
	input := chproto.Input{
		{Name: "user_id", Data: userId},
		{Name: "currency_from", Data: currencyFrom},
		{Name: "currency_to", Data: currencyTo},
		{Name: "amount_sell_micros", Data: amountSellMicros},
		{Name: "amount_buy_micros", Data: amountBuyMicros},
		{Name: "rate", Data: rate},
		{Name: "originating_country", Data: originatingCountry},
		{Name: "time_placed", Data: timePlaced},
	}
	q := ch.Query{
		Body:  input.Into("trades"),
		Input: input,
	}
	if err := s.db.Do(ctx, q); err != nil {
		return fmt.Errorf("failed to save a trade in the DB: %s", err)
	}
	return nil
}
