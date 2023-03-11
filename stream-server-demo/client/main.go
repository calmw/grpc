package main

import (
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Must specify a gRPC server address")
	}
	conn, err := setupGrpcConnection(os.Args[1])
	if err != nil {
		fmt.Println(1)
		log.Fatal(err)
	}
	defer conn.Close()
	c := getRepoServiceClient(conn)
	stream, err := c.GetRepos(context.Background(), &svc.RepoGetRequest{
		Id:        "1",
		CreatorId: "1",
	})
	s := status.Convert(err) // status.Convert函数分别访问错误代码和错误消息
	if s.Code() != codes.OK {
		log.Fatalf("Request failed: %v-%v\n", s.Code(), s.Message())
	}

	var repos []*svc.Repository
	for {
		repo, err := stream.Recv()
		if err == io.EOF {
			break
		}
		s = status.Convert(err) // status.Convert函数分别访问错误代码和错误消息
		if s.Code() != codes.OK {
			log.Fatalf("Request stream failed: %v-%v\n", s.Code(), s.Message())
		}

		fmt.Fprintf(
			os.Stdout,
			"Repos: %v \n",
			repo.Repo,
		)

		repos = append(repos, repo.Repo)
	}

	if len(repos) != 5 {
		fmt.Fprintf(
			os.Stdout,
			"Expected to get back 5 repos, got back: %d repos \n",
			len(repos),
		)
		return
	}

	fmt.Fprintf(
		os.Stdout,
		"Repos: %v \n",
		repos,
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

func getRepoServiceClient(conn *grpc.ClientConn) svc.RepoClient {
	return svc.NewRepoClient(conn)
}
