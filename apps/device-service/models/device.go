package models

import "time"

type Location struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type Device struct {
	ID         int       `db:"id"`
	Name       string    `db:"name"`
	Type       string    `db:"type"`
	Unit       string    `db:"unit"`
	Status     string    `db:"status"`
	LocationID int       `db:"location_id"`
	Location   string    `db:"location_name"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

type DeviceCreate struct {
	Name     string
	Type     string
	Unit     string
	Location string
}

type DeviceUpdate struct {
	Name   string
	Status string
}
