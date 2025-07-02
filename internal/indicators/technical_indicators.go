package indicators

import "time"

type PriceData struct {
	Price     float64
	Volume    float64
	Timestamp time.Time
}
