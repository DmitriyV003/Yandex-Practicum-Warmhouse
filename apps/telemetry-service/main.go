package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"telemetry-service/broker"
	"telemetry-service/clients"
	"telemetry-service/db"
	pb "telemetry-service/proto"
	"telemetry-service/services"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/telemetry_db")
	mqURL := getEnv("RABBITMQ_URL", "amqp://rabbit:rabbit@localhost:5672/")
	tempURL := getEnv("TEMPERATURE_API_URL", "http://temperature-api:8081")
	port := getEnv("PORT", ":8083")

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

	tempClient := clients.NewTemperatureClient(tempURL)
	log.Printf("Temperature client initialized: %s", tempURL)

	svc := &services.TelemetryServiceServer{
		DB:         database,
		Broker:     mq,
		TempClient: tempClient,
	}

	if err := mq.Consume(svc.HandleDeviceEvent); err != nil {
		log.Fatalf("broker consume: %v", err)
	}
	log.Println("Started consuming device events")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTelemetryServiceServer(grpcServer, svc)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("telemetry-service gRPC listening on %s", port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down telemetry-service...")
	grpcServer.GracefulStop()
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
