package models

import "time"

type TelemetryRecord struct {
	ID        int       `db:"id"`
	SensorID  int       `db:"sensor_id"`
	Value     float64   `db:"value"`
	Unit      string    `db:"unit"`
	CreatedAt time.Time `db:"created_at"`
}

type TelemetryCreate struct {
	SensorID int
	Value    float64
	Unit     string
}
