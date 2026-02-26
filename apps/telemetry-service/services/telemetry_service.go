package services

import (
	"context"
	"fmt"
	"log"

	"telemetry-service/broker"
	"telemetry-service/clients"
	"telemetry-service/db"
	"telemetry-service/models"
	pb "telemetry-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TelemetryServiceServer struct {
	pb.UnimplementedTelemetryServiceServer
	DB         *db.DB
	Broker     *broker.Broker
	TempClient *clients.TemperatureClient
}

func (s *TelemetryServiceServer) RecordTelemetry(ctx context.Context, req *pb.RecordTelemetryRequest) (*pb.TelemetryResponse, error) {
	record, err := s.DB.SaveTelemetry(ctx, models.TelemetryCreate{
		SensorID: int(req.SensorId),
		Value:    req.Value,
		Unit:     req.Unit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save telemetry: %v", err)
	}

	s.Broker.Publish(ctx, "telemetry.updated", map[string]interface{}{
		"device_id": req.SensorId,
		"value":     record.Value,
		"unit":      record.Unit,
	})

	return telemetryToProto(record), nil
}

func (s *TelemetryServiceServer) GetLatestTelemetry(ctx context.Context, req *pb.GetTelemetryRequest) (*pb.TelemetryResponse, error) {
	record, err := s.DB.GetLatestBySensorID(ctx, int(req.SensorId))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "telemetry not found: %v", err)
	}
	return telemetryToProto(record), nil
}

func (s *TelemetryServiceServer) ListTelemetry(ctx context.Context, req *pb.ListTelemetryRequest) (*pb.ListTelemetryResponse, error) {
	records, err := s.DB.ListBySensorID(ctx, int(req.SensorId), int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list telemetry: %v", err)
	}

	resp := &pb.ListTelemetryResponse{}
	for _, r := range records {
		resp.Records = append(resp.Records, telemetryToProto(r))
	}
	return resp, nil
}

func (s *TelemetryServiceServer) CollectFromSensor(ctx context.Context, req *pb.CollectFromSensorRequest) (*pb.TelemetryResponse, error) {
	tempResp, err := s.TempClient.GetByID(fmt.Sprintf("%d", req.SensorId))
	if err != nil {
		s.Broker.Publish(ctx, "telemetry.no_data", map[string]interface{}{
			"device_id": req.SensorId,
			"location":  req.Location,
		})
		return nil, status.Errorf(codes.Unavailable, "temperature fetch failed: %v", err)
	}

	record, err := s.DB.SaveTelemetry(ctx, models.TelemetryCreate{
		SensorID: int(req.SensorId),
		Value:    tempResp.Value,
		Unit:     tempResp.Unit,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save telemetry: %v", err)
	}

	s.Broker.Publish(ctx, "telemetry.updated", map[string]interface{}{
		"device_id": req.SensorId,
		"value":     record.Value,
		"unit":      record.Unit,
	})
	log.Printf("collected telemetry for sensor %d: %.2f %s", req.SensorId, record.Value, record.Unit)

	return telemetryToProto(record), nil
}

func (s *TelemetryServiceServer) HandleDeviceEvent(event broker.Event) {
	ctx := context.Background()
	deviceID, ok := event.Payload["device_id"].(float64)
	if !ok {
		log.Printf("invalid device_id in device event: %v", event.Payload)
		return
	}

	switch event.Type {
	case "device.deleted":
		if err := s.DB.DeleteBySensorID(ctx, int(deviceID)); err != nil {
			log.Printf("delete telemetry for device %d: %v", int(deviceID), err)
		} else {
			log.Printf("deleted telemetry records for device %d", int(deviceID))
		}
	case "device.created":
		log.Printf("device %d created, ready to collect telemetry", int(deviceID))
	}
}

func telemetryToProto(r models.TelemetryRecord) *pb.TelemetryResponse {
	return &pb.TelemetryResponse{
		Id:        int32(r.ID),
		SensorId:  int32(r.SensorID),
		Value:     r.Value,
		Unit:      r.Unit,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
}
