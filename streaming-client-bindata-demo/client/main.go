package main

import (
	"bufio"
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
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
		log.Fatal(err)
	}
	defer conn.Close()
	c := getRepoServiceClient(conn)
	err = createRepo(os.Stdin, os.Stdout, c)
	if err != nil {
		log.Fatal(err)
	}

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

func createRepo(stdin io.Reader, stdout io.Writer, c svc.RepoClient) error {
	// 获取要传输的文件名
	scanner := bufio.NewScanner(stdin)
	prompt := "Filename: "
	fmt.Fprint(stdout, prompt)

	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return err
	}
	filename := scanner.Text()

	//
	stream, err := c.CreateRepo(context.Background())
	if err != nil {
		return err
	}
	// 第一步，发送一个仅包含context字段集的RepoCreateContext对象
	requestContext := svc.RepoCreateRequest_Context{Context: &svc.RepoContext{
		CreatorId: "user-123",
		Name:      "test-repo",
	}}
	request := svc.RepoCreateRequest{Body: &requestContext}
	err = stream.Send(&request)
	if err != nil {
		return err
	}
	// 第二步，发送要在存储库中创建的数据
	// 创建句柄
	fi, err := os.Open(filename)
	if err != nil {
		return err
	}

	//创建reader
	r := bufio.NewReader(fi)
	//创建buffer，每次读取1024个字节
	buf := make([]byte, 512) // 为了测试，设置较小缓冲区
	var expectSize int64
	var sss int

	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		err = stream.Send(&svc.RepoCreateRequest{Body: &svc.RepoCreateRequest_Data{Data: buf}})
		if err != nil {
			return err
		}
		sss++
		log.Println("sending repo create data", sss)
		expectSize += int64(len(buf))
	}

	// 第三步从服务器读取响应并验证响应是否包含预期的数据
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Println(expectSize)
	log.Println(resp.Size)
	log.Println(resp.Repo)

	return nil
}
