package models

import "time"

type TelemetryRecord struct {
	ID        int       `db:"id"`
	DeviceID  int       `db:"device_id"`
	Value     float64   `db:"value"`
	Unit      string    `db:"unit"`
	CreatedAt time.Time `db:"created_at"`
}

type TelemetryCreate struct {
	DeviceID int
	Value    float64
	Unit     string
}
