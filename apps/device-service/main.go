package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"device-service/broker"
	"device-service/db"
	pb "device-service/proto"
	"device-service/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/device_db")
	mqURL := getEnv("RABBITMQ_URL", "amqp://rabbit:rabbit@localhost:5672/")
	port := getEnv("PORT", ":8082")

	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()
	log.Println("Connected to database")

	mq, err := broker.NewWithRetry(mqURL, 10)
	if err != nil {
		log.Fatalf("rabbitmq connect: %v", err)
	}
	defer mq.Close()
	log.Println("Connected to RabbitMQ")

	svc := &services.DeviceServiceServer{DB: database, Broker: mq}

	if err := mq.Consume(svc.HandleTelemetryEvent); err != nil {
		log.Fatalf("broker consume: %v", err)
	}
	log.Println("Started consuming telemetry events")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDeviceServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("device-service gRPC listening on %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down device-service...")
	grpcServer.GracefulStop()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
