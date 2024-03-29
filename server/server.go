package main

import (
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	healthsvc "google.golang.org/grpc/health"
	healthz "google.golang.org/grpc/health/grpc_health_v1"
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

const RecvMsgTimeout = time.Millisecond * 500

func main() {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if len(listenAddr) == 0 {
		listenAddr = "localhost:50051"
	}
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	// 获取TLS密钥和证书
	tlsCertFile, ok := os.LookupEnv("TLS_CERT_FILE")
	if !ok {
		tlsCertFile = "./server.crt"
	}
	tlsKeyFile, ok := os.LookupEnv("TLS_KEY_FILE")
	if !ok {
		tlsKeyFile = "./server.key"
	}

	creds, err := credentials.NewServerTLSFromFile(
		tlsCertFile,
		tlsKeyFile,
	)
	if err != nil {
		log.Fatal(err)
	}
	credsOption := grpc.Creds(creds)
	s := grpc.NewServer(
		credsOption,
		grpc.ChainUnaryInterceptor( // 用于注册多个服务端一元拦截器，最内层的拦截器首先执行
			loggingUnaryInterceptor,
			timeoutUnaryInterceptor,
			panicUnaryInterceptor, //
			// ... 其他拦截器
		),
		grpc.ChainStreamInterceptor( // 用于注册多个服务端流拦截器，最内层的拦截器首先执行
			loggingStreamInterceptor,
			timeoutStreamInterceptor,
			panicStreamInterceptor,
			// ... 其他拦截器
		),
	)

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

func stopServer(s *grpc.Server, h *healthsvc.Server, d time.Duration) {
	updateServiceHealth(h, svc.Users_ServiceDesc.ServiceName, healthz.HealthCheckResponse_NOT_SERVING)
	time.Sleep(d)
	s.Stop()
	s.GracefulStop()
}

func updateServiceHealth(
	h *healthsvc.Server,
	service string,
	status healthz.HealthCheckResponse_ServingStatus,
) {
	h.SetServingStatus(service, status)
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

		if request.Request == "panic" {
			panic("I was asked to panic")
		}

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
	if components[0] == "panic" {
		panic("I was asked to panic")
	}
	u := svc.User{
		Id:        in.Id,
		FirstName: components[0],
		LastName:  components[1],
		Age:       36,
	}
	//time.Sleep(time.Second)  // 测试服务端超时
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

// 服务端，一元紧急处理拦截器
func panicUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
			err = status.Error(
				codes.Internal,
				"Unexpected error happened",
			)
		}
	}()
	resp, err = handler(ctx, req)

	return
}

// 服务端，一元超时终止请求拦截器
// 假设我们想对RPC方法执行时间施加一个上限，我们知道对于某些恶意请求用户，RPC调用方法可能需要比300毫秒更长的时间，这种情况下我们只想终止请求
// 任何超过300毫秒的RPC方法都将被终止
func timeoutUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	var resp interface{}
	var err error

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	ch := make(chan error)
	go func() {
		resp, err = handler(ctxWithTimeout, req)
		ch <- err
	}()

	select {
	case <-ctxWithTimeout.Done():
		cancel()
		err = status.Error(
			codes.DeadlineExceeded,
			fmt.Sprintf("%s: DeadlineExceeded", info.FullMethod),
		)
		return resp, err
	case <-ch:
	}

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
	serverStream := wrappedServerStream{ServerStream: stream, RecvMsgTimeout: RecvMsgTimeout}
	err := handler(srv, serverStream)
	ctx := stream.Context()
	logMessage(ctx, info.FullMethod, time.Since(start), err)

	return err
}

// 服务端，流紧急处理拦截器
func panicStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
			err = status.Error(
				codes.Internal,
				"Unexpected error happened",
			)
		}
	}()
	serverStream := wrappedServerStream{ServerStream: stream, RecvMsgTimeout: RecvMsgTimeout}
	err = handler(srv, serverStream)

	return
}

// 服务端，流超时处理拦截器
// wrappedServerStream以及RecvMs是包装和处理过的
func timeoutStreamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
			err = status.Error(
				codes.Internal,
				"Unexpected error happened",
			)
		}
	}()
	serverStream := wrappedServerStream{ServerStream: stream, RecvMsgTimeout: RecvMsgTimeout}
	err = handler(srv, serverStream)

	return
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

// 下面的一个结构体以及方法，是对服务端流的包装，将使用这些方法对原本流处理方法进行替换，来对服务端流的包装，实现每次流传输都可以进行自定义操作，而不是原本的等到全部传输完成才执行拦截器
type wrappedServerStream struct {
	RecvMsgTimeout time.Duration // 流超时时间
	grpc.ServerStream
}

func (s wrappedServerStream) SendMsg(m interface{}) error {
	log.Printf("Send msg called: %T", m)
	return s.ServerStream.SendMsg(m)
}

func (s wrappedServerStream) RecvMsg(m interface{}) error {
	ch := make(chan error)
	t := time.NewTimer(s.RecvMsgTimeout)
	go func() {
		log.Printf("Waiting to receive a msg: %T", m)
		ch <- s.ServerStream.RecvMsg(m)
	}()
	select {
	case <-t.C:
		return status.Error(
			codes.DeadlineExceeded,
			"Deadline exceeded",
		)
	case err := <-ch:
		return err
	}
}
