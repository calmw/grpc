package main

import (
	"context"
	"fmt"
	users "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Must specify a gRPC server address and search query")
	}

	serverAddr := os.Args[1]
	u, err := createUserRequest(os.Args[2])
	conn, err := setupGrpcConnection(serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	c := getUserServiceClient(conn)
	result, err := getUser(c, u)
	if err != nil {
		log.Fatal(err)
	}
	data, err := getUserResponseJson(result)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(
		os.Stdout,
		string(data)+"\n",
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

func createUserRequest(jsonQuery string) (*users.UserGetRequest, error) {
	u := users.UserGetRequest{}
	input := []byte(jsonQuery)
	return &u, protojson.Unmarshal(input, &u)
}

func getUserResponseJson(result *users.UserGetReply) ([]byte, error) {
	return protojson.Marshal(result)
}
