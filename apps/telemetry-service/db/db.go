package db

import (
	"context"
	"fmt"

	"telemetry-service/models"

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

func (db *DB) SaveTelemetry(ctx context.Context, t models.TelemetryCreate) (models.TelemetryRecord, error) {
	var record models.TelemetryRecord
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO telemetry_records (sensor_id, value, unit)
		 VALUES ($1, $2, $3)
		 RETURNING id, sensor_id, value, unit, created_at`,
		t.SensorID, t.Value, t.Unit,
	).Scan(&record.ID, &record.SensorID, &record.Value, &record.Unit, &record.CreatedAt)
	if err != nil {
		return models.TelemetryRecord{}, fmt.Errorf("save telemetry: %w", err)
	}
	return record, nil
}

func (db *DB) GetLatestBySensorID(ctx context.Context, sensorID int) (models.TelemetryRecord, error) {
	var record models.TelemetryRecord
	err := db.Pool.QueryRow(ctx,
		`SELECT id, sensor_id, value, unit, created_at
		 FROM telemetry_records
		 WHERE sensor_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`, sensorID,
	).Scan(&record.ID, &record.SensorID, &record.Value, &record.Unit, &record.CreatedAt)
	if err != nil {
		return models.TelemetryRecord{}, fmt.Errorf("get latest telemetry: %w", err)
	}
	return record, nil
}

func (db *DB) ListBySensorID(ctx context.Context, sensorID, limit int) ([]models.TelemetryRecord, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, sensor_id, value, unit, created_at
		 FROM telemetry_records
		 WHERE sensor_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, sensorID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list telemetry: %w", err)
	}
	defer rows.Close()

	var records []models.TelemetryRecord
	for rows.Next() {
		var r models.TelemetryRecord
		if err := rows.Scan(&r.ID, &r.SensorID, &r.Value, &r.Unit, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan telemetry: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (db *DB) DeleteBySensorID(ctx context.Context, sensorID int) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM telemetry_records WHERE sensor_id = $1", sensorID)
	if err != nil {
		return fmt.Errorf("delete telemetry: %w", err)
	}
	return nil
}
