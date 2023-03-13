package main

import (
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	healthsvc "google.golang.org/grpc/health"
	healthz "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
)

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if len(listenAddr) == 0 {
		listenAddr = "localhost:50051"
	}
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	h := healthsvc.NewServer()
	registerServices(s, h)
	updateServiceHealth(h, svc.Users_ServiceDesc.ServiceName, healthz.HealthCheckResponse_SERVING)
	log.Fatal(startServer(s, lis))
}

type userService struct {
	svc.UnimplementedUsersServer // 对于grpc中任何服务实现都是强制性的
}

func registerServices(s *grpc.Server, h *healthsvc.Server) {
	svc.RegisterUsersServer(s, &userService{})
	healthz.RegisterHealthServer(s, h)
	reflection.Register(s)
}

func startServer(s *grpc.Server, l net.Listener) error {
	return s.Serve(l)
}

func updateServiceHealth(
	h *healthsvc.Server,
	service string,
	status healthz.HealthCheckResponse_ServingStatus,
) {
	h.SetServingStatus(service, status)
}
