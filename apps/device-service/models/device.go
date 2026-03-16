package models

import "time"

type User struct {
	ID           int       `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

type House struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Address   string    `db:"address"`
	OwnerID   int       `db:"owner_id"`
	CreatedAt time.Time `db:"created_at"`
}

type Room struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	HouseID int    `db:"house_id"`
}

type Device struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Type      string    `db:"type"`
	Unit      string    `db:"unit"`
	Status    string    `db:"status"`
	RoomID    int       `db:"room_id"`
	RoomName  string    `db:"room_name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type DeviceCreate struct {
	Name   string
	Type   string
	Unit   string
	RoomID int
}

type DeviceUpdate struct {
	Name   string
	Status string
}

type Scenario struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	HouseID   int       `db:"house_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
}

type ScenarioAction struct {
	ID         int    `db:"id"`
	ScenarioID int    `db:"scenario_id"`
	DeviceID   int    `db:"device_id"`
	Action     string `db:"action"`
	Value      string `db:"value"`
}
