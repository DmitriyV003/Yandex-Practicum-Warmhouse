package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"smarthome/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(connString string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// ---- Devices ----

func (db *DB) GetDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT d.id, d.name, d.type, d.unit, d.status, d.room_id,
		        r.name AS room_name, d.created_at, d.updated_at
		 FROM devices d
		 JOIN rooms r ON r.id = d.room_id
		 ORDER BY d.id`)
	if err != nil {
		return nil, fmt.Errorf("error querying devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Unit, &d.Status,
			&d.RoomID, &d.RoomName, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning device row: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (db *DB) GetDeviceByID(ctx context.Context, id int) (models.Device, error) {
	var d models.Device
	err := db.Pool.QueryRow(ctx,
		`SELECT d.id, d.name, d.type, d.unit, d.status, d.room_id,
		        r.name AS room_name, d.created_at, d.updated_at
		 FROM devices d
		 JOIN rooms r ON r.id = d.room_id
		 WHERE d.id = $1`, id,
	).Scan(&d.ID, &d.Name, &d.Type, &d.Unit, &d.Status,
		&d.RoomID, &d.RoomName, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return models.Device{}, fmt.Errorf("error getting device by ID: %w", err)
	}
	return d, nil
}

func (db *DB) CreateDevice(ctx context.Context, d models.DeviceCreate) (models.Device, error) {
	now := time.Now()
	var device models.Device
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO devices (name, type, unit, status, room_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'inactive', $4, $5, $5)
		 RETURNING id, name, type, unit, status, room_id, created_at, updated_at`,
		d.Name, d.Type, d.Unit, d.RoomID, now,
	).Scan(&device.ID, &device.Name, &device.Type, &device.Unit, &device.Status,
		&device.RoomID, &device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return models.Device{}, fmt.Errorf("error creating device: %w", err)
	}

	_ = db.Pool.QueryRow(ctx, `SELECT name FROM rooms WHERE id = $1`, d.RoomID).Scan(&device.RoomName)

	return device, nil
}

func (db *DB) UpdateDevice(ctx context.Context, id int, u models.DeviceUpdate) (models.Device, error) {
	_, err := db.GetDeviceByID(ctx, id)
	if err != nil {
		return models.Device{}, err
	}

	query := "UPDATE devices SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if u.Name != "" {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, u.Name)
		argCount++
	}
	if u.Type != "" {
		query += fmt.Sprintf(", type = $%d", argCount)
		args = append(args, u.Type)
		argCount++
	}
	if u.Unit != "" {
		query += fmt.Sprintf(", unit = $%d", argCount)
		args = append(args, u.Unit)
		argCount++
	}
	if u.Status != "" {
		query += fmt.Sprintf(", status = $%d", argCount)
		args = append(args, u.Status)
		argCount++
	}

	query += fmt.Sprintf(` WHERE id = $%d
		RETURNING id, name, type, unit, status, room_id, created_at, updated_at`, argCount)
	args = append(args, id)

	var device models.Device
	err = db.Pool.QueryRow(ctx, query, args...).Scan(
		&device.ID, &device.Name, &device.Type, &device.Unit, &device.Status,
		&device.RoomID, &device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return models.Device{}, fmt.Errorf("error updating device: %w", err)
	}

	_ = db.Pool.QueryRow(ctx, `SELECT name FROM rooms WHERE id = $1`, device.RoomID).Scan(&device.RoomName)

	return device, nil
}

func (db *DB) DeleteDevice(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, "DELETE FROM devices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("error deleting device: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("device not found")
	}
	return nil
}

func (db *DB) UpdateDeviceStatus(ctx context.Context, id int, status string) error {
	result, err := db.Pool.Exec(ctx,
		`UPDATE devices SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating device status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("device not found")
	}
	return nil
}

// ---- Rooms ----

func (db *DB) GetRooms(ctx context.Context, houseID int) ([]models.Room, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, house_id FROM rooms WHERE house_id = $1 ORDER BY id`, houseID)
	if err != nil {
		return nil, fmt.Errorf("error querying rooms: %w", err)
	}
	defer rows.Close()

	var rooms []models.Room
	for rows.Next() {
		var r models.Room
		if err := rows.Scan(&r.ID, &r.Name, &r.HouseID); err != nil {
			return nil, fmt.Errorf("error scanning room: %w", err)
		}
		rooms = append(rooms, r)
	}
	return rooms, rows.Err()
}

func (db *DB) CreateRoom(ctx context.Context, r models.RoomCreate) (models.Room, error) {
	var room models.Room
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO rooms (name, house_id) VALUES ($1, $2) RETURNING id, name, house_id`,
		r.Name, r.HouseID,
	).Scan(&room.ID, &room.Name, &room.HouseID)
	if err != nil {
		return models.Room{}, fmt.Errorf("error creating room: %w", err)
	}
	return room, nil
}

// ---- Houses ----

func (db *DB) GetHouses(ctx context.Context, ownerID int) ([]models.House, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, address, owner_id, created_at FROM houses WHERE owner_id = $1 ORDER BY id`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error querying houses: %w", err)
	}
	defer rows.Close()

	var houses []models.House
	for rows.Next() {
		var h models.House
		if err := rows.Scan(&h.ID, &h.Name, &h.Address, &h.OwnerID, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning house: %w", err)
		}
		houses = append(houses, h)
	}
	return houses, rows.Err()
}

func (db *DB) CreateHouse(ctx context.Context, h models.HouseCreate) (models.House, error) {
	var house models.House
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO houses (name, address, owner_id, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, name, address, owner_id, created_at`,
		h.Name, h.Address, h.OwnerID,
	).Scan(&house.ID, &house.Name, &house.Address, &house.OwnerID, &house.CreatedAt)
	if err != nil {
		return models.House{}, fmt.Errorf("error creating house: %w", err)
	}
	return house, nil
}

// ---- Users ----

func (db *DB) CreateUser(ctx context.Context, u models.UserCreate) (models.User, error) {
	var user models.User
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash, created_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, name, email, password_hash, created_at`,
		u.Name, u.Email, u.Password,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("error creating user: %w", err)
	}
	return user, nil
}

func (db *DB) GetUserByID(ctx context.Context, id int) (models.User, error) {
	var u models.User
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("error getting user: %w", err)
	}
	return u, nil
}
