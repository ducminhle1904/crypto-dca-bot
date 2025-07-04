package types

import "time"

type OHLCV struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}

type Ticker struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
}

type Balance struct {
	Asset  string
	Free   float64
	Locked float64
}
