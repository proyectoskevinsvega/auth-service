package domain

import "time"

type Geolocation struct {
	IP        string
	Country   string
	City      string
	Latitude  float64
	Longitude float64
	UpdatedAt time.Time
}
