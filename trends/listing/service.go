package listing

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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

func NewService(db driver.Conn) Service {
	return &service{
		db: db,
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
