package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserCreate struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type House struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type HouseCreate struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
	OwnerID int    `json:"owner_id" binding:"required"`
}

type Room struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	HouseID int    `json:"house_id"`
}

type RoomCreate struct {
	Name    string `json:"name" binding:"required"`
	HouseID int    `json:"house_id" binding:"required"`
}

// DeviceType represents the type of device
type DeviceType string

const (
	Temperature DeviceType = "temperature"
)

// Device represents a smart home device
type Device struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Type      DeviceType `json:"type"`
	Unit      string     `json:"unit"`
	Status    string     `json:"status"`
	RoomID    int        `json:"room_id"`
	RoomName  string     `json:"room_name,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// DeviceCreate represents the data needed to create a new device
type DeviceCreate struct {
	Name   string     `json:"name" binding:"required"`
	Type   DeviceType `json:"type" binding:"required"`
	RoomID int        `json:"room_id" binding:"required"`
	Unit   string     `json:"unit"`
}

// DeviceUpdate represents the data that can be updated for a device
type DeviceUpdate struct {
	Name   string     `json:"name"`
	Type   DeviceType `json:"type"`
	Unit   string     `json:"unit"`
	Status string     `json:"status"`
}

type Scenario struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	HouseID   int       `json:"house_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type ScenarioCreate struct {
	Name    string `json:"name" binding:"required"`
	HouseID int    `json:"house_id" binding:"required"`
}

type ScenarioAction struct {
	ID         int    `json:"id"`
	ScenarioID int    `json:"scenario_id"`
	DeviceID   int    `json:"device_id"`
	Action     string `json:"action"`
	Value      string `json:"value"`
}

type ScenarioActionCreate struct {
	DeviceID int    `json:"device_id" binding:"required"`
	Action   string `json:"action" binding:"required"`
	Value    string `json:"value" binding:"required"`
}
