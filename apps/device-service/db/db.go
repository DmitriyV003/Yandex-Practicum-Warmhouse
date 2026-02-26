package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"device-service/models"

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

func (db *DB) GetOrCreateLocation(ctx context.Context, name string) (int, error) {
	var id int
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO locations (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
		name,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get or create location: %w", err)
	}
	return id, nil
}

func (db *DB) CreateDevice(ctx context.Context, d models.DeviceCreate) (models.Device, error) {
	locationID, err := db.GetOrCreateLocation(ctx, d.Location)
	if err != nil {
		return models.Device{}, err
	}

	now := time.Now()
	var device models.Device
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO devices (name, type, unit, status, location_id, created_at, updated_at)
		 VALUES ($1, $2, $3, 'inactive', $4, $5, $5)
		 RETURNING id, name, type, unit, status, location_id, created_at, updated_at`,
		d.Name, d.Type, d.Unit, locationID, now,
	).Scan(&device.ID, &device.Name, &device.Type, &device.Unit, &device.Status,
		&device.LocationID, &device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return models.Device{}, fmt.Errorf("create device: %w", err)
	}
	device.Location = d.Location
	return device, nil
}

func (db *DB) GetDevice(ctx context.Context, id int) (models.Device, error) {
	var d models.Device
	err := db.Pool.QueryRow(ctx,
		`SELECT d.id, d.name, d.type, d.unit, d.status, d.location_id,
		        l.name AS location_name, d.created_at, d.updated_at
		 FROM devices d
		 JOIN locations l ON l.id = d.location_id
		 WHERE d.id = $1`, id,
	).Scan(&d.ID, &d.Name, &d.Type, &d.Unit, &d.Status, &d.LocationID,
		&d.Location, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return models.Device{}, fmt.Errorf("get device: %w", err)
	}
	return d, nil
}

func (db *DB) ListDevices(ctx context.Context, status, location string) ([]models.Device, error) {
	query := `SELECT d.id, d.name, d.type, d.unit, d.status, d.location_id,
	                 l.name AS location_name, d.created_at, d.updated_at
	          FROM devices d
	          JOIN locations l ON l.id = d.location_id
	          WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		query += fmt.Sprintf(" AND d.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if location != "" {
		query += fmt.Sprintf(" AND l.name = $%d", argIdx)
		args = append(args, location)
		argIdx++
	}
	query += " ORDER BY d.id"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Type, &d.Unit, &d.Status, &d.LocationID,
			&d.Location, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (db *DB) UpdateDevice(ctx context.Context, id int, u models.DeviceUpdate) (models.Device, error) {
	query := "UPDATE devices SET updated_at = $1"
	args := []interface{}{time.Now()}
	argIdx := 2

	if u.Name != "" {
		query += fmt.Sprintf(", name = $%d", argIdx)
		args = append(args, u.Name)
		argIdx++
	}
	if u.Status != "" {
		query += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, u.Status)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argIdx)
	args = append(args, id)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return models.Device{}, fmt.Errorf("update device: %w", err)
	}
	return db.GetDevice(ctx, id)
}

func (db *DB) DeleteDevice(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, "DELETE FROM devices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errors.New("device not found")
	}
	return nil
}
