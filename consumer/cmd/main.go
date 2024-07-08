package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/anton-kapralov/currency-market-pulse/consumer/http/rest"
)

func main() {
	router := gin.Default()
	router.POST("/api/trade", rest.Consume)

	log.Fatalln(router.Run(":8081"))
}
