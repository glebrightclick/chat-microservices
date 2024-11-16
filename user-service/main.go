package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"user-service/controllers"
	"user-service/database"
	"user-service/proto"
	"user-service/repositories"
)

func main() {
	server := grpc.NewServer()
	database.ConnectDatabase()

	repository := &repositories.GormUserRepository{}
	userServiceServer := &controllers.UserServiceServer{UserRepo: repository}
	proto.RegisterUserServiceServer(server, userServiceServer)

	// Enable gRPC reflection
	reflection.Register(server)

	listener, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("failed to listen on port 50051: %v", err)
	}

	log.Println("gRPC server is running on port 50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}

}
