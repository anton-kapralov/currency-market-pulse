package rest

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
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

func Consume(c *gin.Context) {
	var msg message
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("%+v", msg)
	c.Status(http.StatusAccepted)
}
