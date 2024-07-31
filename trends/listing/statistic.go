package listing

import "time"

type Statistic struct {
	Window time.Time
	Min    float64
	Max    float64
	Mean   float64
	Median float64
}
