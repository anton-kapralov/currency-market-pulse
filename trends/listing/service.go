package listing

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/bradfitz/gomemcache/memcache"
)

type Service interface {
	List(
		ctx context.Context,
		dateFrom, dateTo time.Time,
		currencyFrom, currencyTo string,
	) ([]Statistic, error)
}

type service struct {
	db driver.Conn
}

type cachedService struct {
	service
	mc *memcache.Client
}

func NewService(db driver.Conn, mc *memcache.Client) Service {
	return &cachedService{
		service: service{
			db: db,
		},
		mc: mc,
	}
}

func (s *service) List(
	ctx context.Context,
	dateFrom, dateTo time.Time,
	currencyFrom, currencyTo string,
) ([]Statistic, error) {
	query := `
SELECT
	toStartOfInterval(toDateTime(t.time_placed), INTERVAL 10 minute) as time_window,
	min(t.rate),
	max(t.rate),
	avg(t.rate),
	median(t.rate)
FROM cmp.trades t
WHERE 
    t.time_placed>=? AND 
    t.time_placed<=? AND 
    t.currency_from=? AND 
    t.currency_to=?
GROUP BY time_window
ORDER BY time_window
`
	rows, err := s.db.Query(ctx, query, dateFrom, dateTo, currencyFrom, currencyTo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from DB: %s", err)
	}
	var res []Statistic
	for rows.Next() {
		var stat Statistic
		if err := rows.Scan(&stat.Window, &stat.Min, &stat.Max, &stat.Mean, &stat.Median); err != nil {
			return nil, fmt.Errorf("failed to scan row: %s", err)
		}
		res = append(res, stat)
	}
	return res, nil
}

func (s *cachedService) List(
	ctx context.Context,
	dateFrom, dateTo time.Time,
	currencyFrom, currencyTo string,
) ([]Statistic, error) {
	key := fmt.Sprintf("%d-%d-%s-%s", dateFrom.UnixMilli(), dateTo.UnixMilli(), currencyFrom, currencyTo)
	cacheItem, err := s.mc.Get(key)
	if err != nil {
		if !errors.Is(err, memcache.ErrCacheMiss) {
			log.Printf("failed to fetch the cache item: %s", err)
			return s.service.List(ctx, dateFrom, dateTo, currencyFrom, currencyTo)
		}
		statistics, err := s.service.List(ctx, dateFrom, dateTo, currencyFrom, currencyTo)
		if err != nil {
			return nil, err
		}
		if err := s.cache(key, statistics); err != nil {
			log.Printf("failed to cache the item: %s", err)
		}
		return statistics, nil
	}

	decoder := gob.NewDecoder(bytes.NewReader(cacheItem.Value))
	var statistics []Statistic
	if err := decoder.Decode(&statistics); err != nil {
		log.Printf("failed to decode the cache item: %s", err)
		return s.service.List(ctx, dateFrom, dateTo, currencyFrom, currencyTo)
	}
	return statistics, nil
}

func (s *cachedService) cache(key string, statistics []Statistic) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(statistics); err != nil {
		return fmt.Errorf("failed to encode the results: %v", err)
	}
	cacheItem := &memcache.Item{
		Key:   key,
		Value: buf.Bytes(),
	}
	if err := s.mc.Set(cacheItem); err != nil {
		return fmt.Errorf("failed to cache the results: %v", err)
	}
	return nil
}
