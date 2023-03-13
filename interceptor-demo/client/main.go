package main

import (
	"bufio"
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Specify a gRPC server and method to call")
	}
	serverAddr := os.Args[1]
	methodName := os.Args[2]

	conn, err := setupGrpcConnection(serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	c := getUserServiceClient(conn)
	switch methodName {
	case "GetUser":

		result, err := c.GetUser(context.Background(), &svc.UserGetRequest{
			Email: "jane@doe.com",
		})

		s := status.Convert(err) // status.Convert函数分别访问错误代码和错误消息
		if s.Code() != codes.OK {
			log.Fatalf("Request failed: %v-%v\n", s.Code(), s.Message())
		}
		fmt.Fprintf(
			os.Stdout,
			"User: %s %s\n",
			result.User.FirstName,
			result.User.LastName,
		)

	case "GetHelp":
		err = setupChat(os.Stdin, os.Stdout, c)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Unrecognized method name")
	}

	if err != nil {
		log.Fatal(err)
	}

}

// 一元客户端拦截器，对传出的任何一元RPC请求添加一个唯一标识符，添加的数据在context中
func metadataUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	ctxWithMetadata := metadata.AppendToOutgoingContext(
		ctx,
		"Request-Id",
		"request-123",
	)
	return invoker(
		ctxWithMetadata,
		method,
		req,
		reply,
		cc,
		opts...,
	)
}

// 流RPC方法调用的元数据拦截器
func metadataStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {

	ctxWithMetadata := metadata.AppendToOutgoingContext(
		ctx,
		"Request-Id",
		"request-123",
	)
	clientStream, err := streamer(
		ctxWithMetadata,
		desc,
		cc,
		method,
		opts...,
	)
	return clientStream, err
}

func setupGrpcConnection(addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithInsecure(), // 与服务器建立非TSL连接
		grpc.WithBlock(),    // 确保在函数返回之前建立连接。这意味着如果在服务器启动并运行之前运行客户端，它将无限期等待。
		grpc.WithUnaryInterceptor(metadataUnaryInterceptor),   // 注册客户端一元拦截器
		grpc.WithStreamInterceptor(metadataStreamInterceptor), // 注册客户端流拦截器
	)
}

func getUserServiceClient(conn *grpc.ClientConn) svc.UsersClient {
	return svc.NewUsersClient(conn)
}

func setupChat(r io.Reader, w io.Writer, c svc.UsersClient) error {

	stream, err := c.GetHelp(context.Background())
	if err != nil {
		return err
	}

	for {
		scanner := bufio.NewScanner(r)
		prompt := "Request: "
		fmt.Fprint(w, prompt)

		scanner.Scan()
		if err = scanner.Err(); err != nil {
			return err
		}
		msg := scanner.Text()
		if msg == "quit" {
			break
		}
		request := svc.UserHelpRequest{
			Request: msg,
		}

		err = stream.Send(&request)
		if err != nil {
			return err
		}

		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		fmt.Printf("Response: %s\n", resp.Response)

	}

	return stream.CloseSend() // 该方法将关闭服务器上的客户端链接，并返回io.EOF错误值
}
