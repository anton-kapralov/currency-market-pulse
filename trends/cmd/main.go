package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"

	"github.com/anton-kapralov/currency-market-pulse/trends/http/rest"
	"github.com/anton-kapralov/currency-market-pulse/trends/listing"
)

func newClickhouseConnection(host string, port int) driver.Conn {
	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("Connecting to Clickhouse at %s", addr)
	ctx := context.Background()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "cmp",
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to Clickhouse at %s: %s", addr, err)
	}
	if err := conn.Ping(ctx); err != nil {
		var exception *clickhouse.Exception
		if errors.As(err, &exception) {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		log.Fatalf("Failed to ping Clickhouse DB: %s", err)
	}
	return conn
}

type options struct {
	clickhouse struct {
		host string
		port int
	}
}

func main() {
	var opts options
	flag.StringVar(&opts.clickhouse.host, "clickhouse.host", "localhost", "Kafka host")
	flag.IntVar(&opts.clickhouse.port, "clickhouse.port", 9000, "Kafka port")
	flag.Parse()

	clickhouseConn := newClickhouseConnection(opts.clickhouse.host, opts.clickhouse.port)
	listingService := listing.NewService(clickhouseConn)

	restController := rest.NewController(listingService)

	router := gin.Default()
	router.GET("/api/trends", restController.Trends)

	log.Fatalln(router.Run(":8082"))
}
