package rest

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anton-kapralov/currency-market-pulse/consumer/consuming"
	"github.com/anton-kapralov/currency-market-pulse/consumer/ratelimit"
	"github.com/anton-kapralov/currency-market-pulse/proto-gen/currencymarketpb"
)

type message struct {
	UserId             string  `json:"userId"`
	CurrencyFrom       string  `json:"currencyFrom"`
	CurrencyTo         string  `json:"currencyTo"`
	AmountSell         float64 `json:"amountSell"`
	AmountBuy          float64 `json:"amountBuy"`
	Rate               float64 `json:"rate"`
	TimePlaced         string  `json:"timePlaced"`
	OriginatingCountry string  `json:"originatingCountry"`
}

type Controller interface {
	SaveTradeMessage(c *gin.Context)
}

type controller struct {
	consumer    consuming.Service
	rateLimiter ratelimit.RateLimiter
}

func NewController(consumer consuming.Service, rateLimiter ratelimit.RateLimiter) Controller {
	return &controller{consumer: consumer, rateLimiter: rateLimiter}
}

func (c *controller) SaveTradeMessage(ctx *gin.Context) {
	var msg message
	if err := ctx.ShouldBindJSON(&msg); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("%+v", msg)

	timeToWait, err := c.rateLimiter.ShouldWait(ctx, msg.UserId)
	if err != nil {
		log.Println(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	if timeToWait > 0 {
		errorMsg := fmt.Sprintf("You must wait %s before you can make requests.", timeToWait)
		ctx.Header("X-Ratelimit-Retry-After", time.Now().Add(timeToWait).UTC().Format(time.RFC3339))
		ctx.JSON(http.StatusTooManyRequests, gin.H{"error": errorMsg})
		return
	}

	timePlaced, err := time.Parse("02-Jan-06 15:04:05", msg.TimePlaced)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	trade := &currencymarketpb.Trade{
		UserId:             msg.UserId,
		CurrencyFrom:       msg.CurrencyFrom,
		CurrencyTo:         msg.CurrencyTo,
		AmountSellMicros:   toMicros(msg.AmountSell),
		AmountBuyMicros:    toMicros(msg.AmountBuy),
		TimePlacedMs:       timePlaced.UnixMilli(),
		OriginatingCountry: msg.OriginatingCountry,
	}
	if err := c.consumer.Save(trade); err != nil {
		log.Println(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusAccepted)
}

func toMicros(amount float64) int64 {
	return int64(amount * 1_000_000)
}
