package services

import (
	"context"
	"log"

	"device-service/broker"
	"device-service/db"
	"device-service/models"
	pb "device-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeviceServiceServer struct {
	pb.UnimplementedDeviceServiceServer
	DB     *db.DB
	Broker *broker.Broker
}

func (s *DeviceServiceServer) CreateDevice(ctx context.Context, req *pb.CreateDeviceRequest) (*pb.DeviceResponse, error) {
	device, err := s.DB.CreateDevice(ctx, models.DeviceCreate{
		Name:     req.Name,
		Type:     req.Type,
		Unit:     req.Unit,
		Location: req.Location,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create device: %v", err)
	}

	s.Broker.Publish(ctx, "device.created", map[string]interface{}{
		"device_id": device.ID,
		"name":      device.Name,
		"type":      device.Type,
		"location":  device.Location,
	})
	log.Printf("device created: id=%d name=%s", device.ID, device.Name)

	return deviceToProto(device), nil
}

func (s *DeviceServiceServer) GetDevice(ctx context.Context, req *pb.GetDeviceRequest) (*pb.DeviceResponse, error) {
	device, err := s.DB.GetDevice(ctx, int(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", err)
	}
	return deviceToProto(device), nil
}

func (s *DeviceServiceServer) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	devices, err := s.DB.ListDevices(ctx, req.Status, req.Location)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list devices: %v", err)
	}

	resp := &pb.ListDevicesResponse{}
	for _, d := range devices {
		resp.Devices = append(resp.Devices, deviceToProto(d))
	}
	return resp, nil
}

func (s *DeviceServiceServer) UpdateDevice(ctx context.Context, req *pb.UpdateDeviceRequest) (*pb.DeviceResponse, error) {
	device, err := s.DB.UpdateDevice(ctx, int(req.Id), models.DeviceUpdate{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update device: %v", err)
	}
	return deviceToProto(device), nil
}

func (s *DeviceServiceServer) DeleteDevice(ctx context.Context, req *pb.DeleteDeviceRequest) (*pb.DeleteDeviceResponse, error) {
	device, err := s.DB.GetDevice(ctx, int(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", err)
	}

	if err := s.DB.DeleteDevice(ctx, int(req.Id)); err != nil {
		return nil, status.Errorf(codes.Internal, "delete device: %v", err)
	}

	s.Broker.Publish(ctx, "device.deleted", map[string]interface{}{
		"device_id": device.ID,
	})
	log.Printf("device deleted: id=%d", device.ID)

	return &pb.DeleteDeviceResponse{Success: true}, nil
}

func (s *DeviceServiceServer) HandleTelemetryEvent(event broker.Event) {
	ctx := context.Background()
	deviceID, ok := event.Payload["device_id"].(float64)
	if !ok {
		log.Printf("invalid device_id in telemetry event: %v", event.Payload)
		return
	}

	newStatus := "active"
	if event.Type == "telemetry.no_data" {
		newStatus = "inactive"
	}

	if _, err := s.DB.UpdateDevice(ctx, int(deviceID), models.DeviceUpdate{Status: newStatus}); err != nil {
		log.Printf("update device status from event: %v", err)
	} else {
		log.Printf("device %d status updated to %s (event: %s)", int(deviceID), newStatus, event.Type)
	}
}

func deviceToProto(d models.Device) *pb.DeviceResponse {
	return &pb.DeviceResponse{
		Id:        int32(d.ID),
		Name:      d.Name,
		Type:      d.Type,
		Unit:      d.Unit,
		Status:    d.Status,
		Location:  d.Location,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
	}
}
