package services

import (
	"context"
	"fmt"

	pb "smarthome/proto/telemetry"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TelemetryClient struct {
	client pb.TelemetryServiceClient
	conn   *grpc.ClientConn
}

func NewTelemetryClient(address string) (*TelemetryClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial telemetry-service: %w", err)
	}
	return &TelemetryClient{client: pb.NewTelemetryServiceClient(conn), conn: conn}, nil
}

func (c *TelemetryClient) Close() {
	c.conn.Close()
}

func (c *TelemetryClient) CollectFromSensor(ctx context.Context, sensorID int32, location string) (*pb.TelemetryResponse, error) {
	return c.client.CollectFromSensor(ctx, &pb.CollectFromSensorRequest{
		SensorId: sensorID,
		Location: location,
	})
}

func (c *TelemetryClient) GetLatest(ctx context.Context, sensorID int32) (*pb.TelemetryResponse, error) {
	return c.client.GetLatestTelemetry(ctx, &pb.GetTelemetryRequest{
		SensorId: sensorID,
	})
}
