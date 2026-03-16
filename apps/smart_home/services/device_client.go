package services

import (
	"context"
	"fmt"

	pb "smarthome/proto/device"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DeviceClient struct {
	client pb.DeviceServiceClient
	conn   *grpc.ClientConn
}

func NewDeviceClient(address string) (*DeviceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial device-service: %w", err)
	}
	return &DeviceClient{client: pb.NewDeviceServiceClient(conn), conn: conn}, nil
}

func (c *DeviceClient) Close() {
	c.conn.Close()
}

func (c *DeviceClient) CreateDevice(ctx context.Context, name, devType, unit string, roomID int32) (*pb.DeviceResponse, error) {
	return c.client.CreateDevice(ctx, &pb.CreateDeviceRequest{
		Name:   name,
		Type:   devType,
		Unit:   unit,
		RoomId: roomID,
	})
}

func (c *DeviceClient) GetDevice(ctx context.Context, id int32) (*pb.DeviceResponse, error) {
	return c.client.GetDevice(ctx, &pb.GetDeviceRequest{Id: id})
}

func (c *DeviceClient) ListDevices(ctx context.Context) ([]*pb.DeviceResponse, error) {
	resp, err := c.client.ListDevices(ctx, &pb.ListDevicesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Devices, nil
}

func (c *DeviceClient) DeleteDevice(ctx context.Context, id int32) error {
	_, err := c.client.DeleteDevice(ctx, &pb.DeleteDeviceRequest{Id: id})
	return err
}
