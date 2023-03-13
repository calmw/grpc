package main

import (
	"context"
	"google.golang.org/grpc"
	healthz "google.golang.org/grpc/health/grpc_health_v1"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Specify a gRPC server")
	}
	serverAddr := os.Args[1]

	healthClient, err := getHealthSvcClient(serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	// 测试 check， 客户端在必要时候可以定时调用Check来检查服务端健康变化
	resp, err := healthClient.Check(
		context.Background(),
		&healthz.HealthCheckRequest{
			Service: "Users", // 如果指定一个没有设置健康状态的服务，将得到一个非nil错误的响应，并且响应的resp为nil，错误响应码将设置为codes.NotFound
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	healthStatus := resp.Status.String()
	if healthStatus != "SERVING" {
		log.Fatalf(
			"Expected status SERVING, got %s",
			healthStatus,
		)
	}
	log.Println("Check, SERVING...")

	// 测试watch， 当服务端更改状态时（例如调用updateServiceHealth更改），客户端就会收到状态变化
	client, err := healthClient.Watch(
		context.Background(),
		&healthz.HealthCheckRequest{
			Service: "Users", // 如果指定一个没有设置健康状态的服务，将得到一个非nil错误的响应，并且响应的resp为nil，错误响应码将设置为codes.NotFound
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for {
		resp, err = client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error in watch: %v \n", err)
		}
		log.Printf("Health status: %#v \n", resp)
		if resp.Status != healthz.HealthCheckResponse_SERVING {
			log.Printf("Unhealthy: %#v \n", resp)
		}
	}

}

func getHealthSvcClient(addr string) (healthz.HealthClient, error) {

	client, err := grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithBlock(),    // 确保在函数返回之前建立连接。这意味着如果在服务器启动并运行之前运行客户端，它将无限期等待。
		grpc.WithInsecure(), // 与服务器建立非TSL连接
	)
	if err != nil {
		return nil, err
	}
	return healthz.NewHealthClient(client), nil
}
