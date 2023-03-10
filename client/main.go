package main

import (
	"context"
	"fmt"
	users "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Must specify a gRPC server address")
	}
	conn, err := setupGrpcConnection(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	c := getUserServiceClient(conn)

	result, err := c.GetUser(context.Background(), &users.UserGetRequest{
		Email: "<?b Start>jane@doe.com<?b End?>",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(
		os.Stdout,
		"User: %s %s\n",
		result.User.FirstName,
		result.User.LastName,
	)
}

func setupGrpcConnection(addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithInsecure(), // 与服务器建立非TSL连接
		grpc.WithBlock(),    // 确保在函数返回之前建立连接。这意味着如果在服务器启动并运行之前运行客户端，它将无限期等待。
	)
}

func getUserServiceClient(conn *grpc.ClientConn) users.UsersClient {
	return users.NewUsersClient(conn)
}

func getUser(client users.UsersClient, u *users.UserGetRequest) (*users.UserGetReply, error) {
	return client.GetUser(context.Background(), u)
}
