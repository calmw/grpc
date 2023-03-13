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
	stream, err := streamer(
		ctxWithMetadata,
		desc,
		cc,
		method,
		opts...,
	)
	clientStream := wrappedClientStream{ClientStream: stream} // 对数据进行包装

	return clientStream, err
}

func setupGrpcConnection(addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithInsecure(), // 与服务器建立非TSL连接
		grpc.WithBlock(),    // 确保在函数返回之前建立连接。这意味着如果在服务器启动并运行之前运行客户端，它将无限期等待。
		grpc.WithChainUnaryInterceptor( // 用于注册多个客户端一元拦截器，最内层的拦截器首先执行
			metadataUnaryInterceptor,
			// ... 其他拦截器
		),
		grpc.WithChainStreamInterceptor( // 用于注册多个客户端流拦截器，最内层的拦截器首先执行
			metadataStreamInterceptor,
			// ... 其他拦截器
		),
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

// 下面的一个结构体以及方法，是对客户端流的包装，将使用这些方法对原本流处理方法进行替换，来对客户端流的包装，实现每次流传输都可以进行自定义操作，而不是原本的等到全部传输完成才执行拦截器
type wrappedClientStream struct {
	grpc.ClientStream
}

func (s wrappedClientStream) SendMsg(m interface{}) error {
	log.Printf("Send msg called: %T", m)
	return s.ClientStream.SendMsg(m)
}

func (s wrappedClientStream) RecvMsg(m interface{}) error {
	log.Printf("Recv msg called: %T", m)
	return s.ClientStream.RecvMsg(m)
}

func (s wrappedClientStream) CloseSend() error {
	log.Printf("CloseSend() called")
	return s.ClientStream.CloseSend()
}
