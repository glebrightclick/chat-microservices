package test

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	_ "google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"user-service/models"
	"user-service/proto"
)

type MockUserRepository struct {
	users map[string]*models.User // Simulates an in-memory storage of users by email
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (m *MockUserRepository) CreateUser(user *models.User) error {
	if _, exists := m.users[user.Name]; exists {
		return errors.New("user already exists")
	}
	m.users[user.Name] = user
	return nil
}

func (m *MockUserRepository) FindUserByName(name string) (*models.User, error) {
	user, exists := m.users[name]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

var listener *bufconn.Listener

func InitGrpcServer(t *testing.T, server proto.UserServiceServer) *grpc.ClientConn {
	listener = bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	proto.RegisterUserServiceServer(s, server)

	go func() {
		if err := s.Serve(listener); err != nil {
			t.Fatalf("Server failed to start: %v", err)
		}
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	return conn
}
