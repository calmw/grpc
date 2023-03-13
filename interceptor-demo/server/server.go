package main

import (
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
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
	s := grpc.NewServer(
		grpc.UnaryInterceptor(loggingUnaryInterceptor),
		grpc.StreamInterceptor(loggingStreamInterceptor),
	)
	registerServer(s)
	log.Fatal(startServer(s, lis))
}

type userService struct {
	svc.UnimplementedUsersServer // 对于grpc中任何服务实现都是强制性的
}

func registerServer(s *grpc.Server) {
	svc.RegisterUsersServer(s, &userService{})
	reflection.Register(s)
}

func startServer(s *grpc.Server, l net.Listener) error {
	return s.Serve(l)
}

func (s *userService) GetHelp(stream svc.Users_GetHelpServer) error {
	log.Println("Client connected")

	for {
		request, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Printf("Request received: %s \n", request.Request)

		response := svc.UserHelpReply{Response: request.Request}

		err = stream.Send(&response)
		if err != nil {
			return err
		}
	}

	log.Println("Client disconnected")
	return nil
}

func (s *userService) GetUser(ctx context.Context, in *svc.UserGetRequest) (*svc.UserGetReply, error) {
	log.Printf(
		"Received request for user with Email: %s Id:%s\n",
		in.Email,
		in.Id,
	)
	components := strings.Split(in.Email, "@")
	if len(components) != 2 {
		return nil, status.Error(codes.InvalidArgument, "Invalid email address specified") // status.Error函数创建错误，可以错误码和错误信息一块创建
	}
	u := svc.User{
		Id:        in.Id,
		FirstName: components[0],
		LastName:  components[1],
		Age:       36,
	}
	return &svc.UserGetReply{User: &u}, nil
}

// 服务端，一元RPC方法调用的日志拦截器
func loggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	logMessage(ctx, info.FullMethod, time.Since(start), err)

	return resp, err
}

// 服务端，流RPC调用的日志拦截器
func loggingStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, stream)
	ctx := stream.Context()
	logMessage(ctx, info.FullMethod, time.Since(start), err)

	return err
}

//记录RPC方法调用的详细信息
func logMessage(
	ctx context.Context,
	method string,
	latency time.Duration,
	err error,
) {
	var requestId string
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("No metadata in context")
	} else {
		if len(md.Get("Request-Id")) != 0 {
			requestId = md.Get("Request-Id")[0]
		}
	}
	log.Printf(
		"Method: %s, Latency: %v, Error: %v, RequestId: %s",
		method,
		latency,
		err,
		requestId,
	)

}
