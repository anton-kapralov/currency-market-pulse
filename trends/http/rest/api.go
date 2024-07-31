package rest

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anton-kapralov/currency-market-pulse/trends/listing"
)

type statisticPage struct {
	DateFrom     string      `json:"dateFrom,omitempty"`
	DateTo       string      `json:"dateTo,omitempty"`
	CurrencyFrom string      `json:"currencyFrom,omitempty"`
	CurrencyTo   string      `json:"currencyTo,omitempty"`
	Statistics   []statistic `json:"statistics,omitempty"`
}

type statistic struct {
	Window time.Time `json:"window"`
	Min    float64   `json:"min,omitempty"`
	Max    float64   `json:"max,omitempty"`
	Mean   float64   `json:"mean,omitempty"`
	Median float64   `json:"median,omitempty"`
}

type Controller interface {
	Trends(c *gin.Context)
}

type controller struct {
	listingService listing.Service
}

func NewController(listingService listing.Service) Controller {
	return &controller{listingService: listingService}
}

func (c *controller) Trends(ctx *gin.Context) {
	dateFromStr := ctx.Query("dateFrom")
	dateToStr := ctx.Query("dateTo")
	currencyFrom := ctx.Query("curFrom")
	currencyTo := ctx.Query("curTo")
	if dateFromStr == "" || dateToStr == "" || currencyFrom == "" || currencyTo == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "required parameters not set"})
		return
	}
	dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
	if err != nil {
		msg := fmt.Sprintf("failed to parse dateFrom: %s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	dateTo, err := time.Parse(time.RFC3339, dateToStr)
	if err != nil {
		msg := fmt.Sprintf("failed to parse dateTo: %s", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	statistics, err := c.listingService.List(ctx, dateFrom, dateTo, currencyFrom, currencyTo)
	if err != nil {
		log.Println(err)
		ctx.Status(http.StatusInternalServerError)
	}
	page := &statisticPage{
		DateFrom:     dateFromStr,
		DateTo:       dateToStr,
		CurrencyFrom: currencyFrom,
		CurrencyTo:   currencyTo,
		Statistics:   make([]statistic, len(statistics)),
	}
	for i, s := range statistics {
		page.Statistics[i] = statistic(s)
	}
	ctx.JSON(http.StatusOK, page)
}
