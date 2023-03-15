package main

import (
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Specify a gRPC server and method to call")
	}
	serverAddr := os.Args[1]
	methodName := os.Args[2]

	// 获取TLS证书
	tlsCertFile, ok := os.LookupEnv("TLS_CERT_FILE")
	if !ok {
		tlsCertFile = "./server.crt"
	}
	conn, cancel, err := setupGrpcConnection(serverAddr, tlsCertFile)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()
	defer conn.Close()
	c := getUserServiceClient(conn)
	switch methodName {
	case "GetUser":
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// 增加 grpc.WaitForReady(true)， 这样在方法调用时，如果未建立到服务器的连接，它会先尝试建立连接，然后调用RPC方法。这样，客户端执行完一次调用，再执行下一次调用前，服务端退出了，下一次调用会会阻塞去连接服务端，连接成功后继续调用，连上之前会一直阻塞。
		result, err := c.GetUser(ctx, &svc.UserGetRequest{
			//Email: "panic@doe.com", // 测试服务端panic @前为panic时触发服务端panic，其它正常
			Email: "cisco@doe.com", // @前为panic时触发服务端panic，其它正常
		}, grpc.WaitForReady(true))

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
		//time.Sleep(time.Second * 2) // 加for循环和等待为了测试grpc.WaitForReady，服务端杀死后，客户端会阻塞等待连接成功后再继续调用
		//}

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

func setupGrpcConnection(addr, tlsCertFile string) (*grpc.ClientConn, context.CancelFunc, error) {
	log.Printf("Connecting to server on %s\n", addr)
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	creds, err := credentials.NewClientTLSFromFile(tlsCertFile, "") // 如果第二个参数非空，将覆盖在证书中找到的主机名。并且该主机名将被信任。我们将为localhost主机名生成TLS证书，这是我们希望客户端信任的主机名，因此指定一个空字符串。
	if err != nil {
		return nil, cancel, err
	}

	credsOption := grpc.WithTransportCredentials(creds)

	// DialContext 在配置这两项 grpc.FailOnNonTempDialError(true), grpc.WithReturnConnectionError()后，将表现出以下行为
	// 1）遇到非临时错误时会立即返回。返回的错误值将包含遇到的错误详细信息。
	// 2）如果遇到非临时错误，它只会尝试建立连接10秒，该函数将返回非临时错误详细信息的错误值
	conn, err := grpc.DialContext(
		ctx,
		addr,
		credsOption,
		grpc.WithBlock(),                  // 确保在函数返回之前建立连接。这意味着如果在服务器启动并运行之前运行客户端，它将无限期等待。即使存在需要检查的永久性故障（例如：指定格式错误的服务器地址活不存在的主机名），这也可能导致客户端继续尝试建立连接而不退出。增加下面选项后，有些情况就不会一直等待，不返回错误
		grpc.FailOnNonTempDialError(true), // true参数，如果发生非临时错误，将不再尝试重新建立连接，DialContext函数将返回遇到的错误
		grpc.WithReturnConnectionError(),  // 使用此选项，当发生临时错误并且上下文在DialContext函数成功之前到期时，返回的错误还将包含阻止连接发生的原始错误。
		grpc.WithChainUnaryInterceptor( // 用于注册多个客户端一元拦截器，最内层的拦截器首先执行
			metadataUnaryInterceptor,
			// ... 其他拦截器
		),
		grpc.WithChainStreamInterceptor( // 用于注册多个客户端流拦截器，最内层的拦截器首先执行
			metadataStreamInterceptor,
			// ... 其他拦截器
		),
	)

	return conn, cancel, err
}

func getUserServiceClient(conn *grpc.ClientConn) svc.UsersClient {
	return svc.NewUsersClient(conn)
}

func createHelpStream(c svc.UsersClient) (svc.Users_GetHelpClient, error) {

	return c.GetHelp(context.Background(), grpc.WaitForReady(true))
}

func setupChat(r io.Reader, w io.Writer, c svc.UsersClient) error {
	var clientConn = make(chan svc.Users_GetHelpClient)
	var done = make(chan struct{})

	stream, err := createHelpStream(c)
	defer stream.CloseSend()
	if err != nil {
		return err
	}

	go func() {
		for {
			clientConn <- stream

			resp, err := stream.Recv()
			if err == io.EOF {
				done <- struct{}{}
			}

			if err != nil {
				stream, err = createHelpStream(c)
				if err != nil {
					log.Printf("Recreating stream failed: %v", err)
					close(clientConn)
					done <- struct{}{}
					break
				}

			} else {
				fmt.Printf("Response: %s\n", resp.Response)
			}

		}
	}()

	requestMsg := "hello"
	msgCount := 1
	for {
		if msgCount > 10 {
			break
		}
		stream = <-clientConn
		if stream == nil {
			break
		}
		if msgCount == 6 {
			time.Sleep(time.Second * 2)
		}

		request := svc.UserHelpRequest{
			Request: fmt.Sprintf("%s-%d", requestMsg, msgCount),
		}

		err = stream.Send(&request)

		if err == io.EOF {
			var m svc.UserGetReply
			err = stream.RecvMsg(&m)
			if err != nil {
				stream, err = createHelpStream(c)
				if err != nil {
					log.Printf("Recreating stream failed: %v", err)
					close(clientConn)
					done <- struct{}{}
					break
				}
			}
			msgCount++
		} else {
			if err != nil {
				log.Printf("Sending request failed: %v. Will retry.\n", err)
			} else {
				log.Printf("Request sent: %s-%d\n", requestMsg, msgCount)
				msgCount++
			}
		}

	}

	<-done
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
