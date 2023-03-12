package main

import (
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"os"
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
	s := grpc.NewServer()
	registerServer(s)
	log.Fatal(startServer(s, lis))
}

type repoService struct {
	svc.UnimplementedRepoServer // 对于grpc中任何服务实现都是强制性的
}

func (s *repoService) CreateRepo(stream svc.Repo_CreateRepoServer) error {
	log.Println("Client connected")

	var repoContext *svc.RepoContext
	var data []byte

	for {
		r, err := stream.Recv()
		fmt.Println(r, "----", err)
		if err == io.EOF {
			break
		}
		if err != nil { // 非读完数据流的其他错误
			return err
		}
		switch t := r.Body.(type) {
		case *svc.RepoCreateRequest_Context:
			repoContext = r.GetContext()
		case *svc.RepoCreateRequest_Data:
			b := r.GetData()
			data = append(data, b...)
		case nil:
			return status.Error(codes.InvalidArgument, "Message doesn't contain context or data")
		default:
			return status.Errorf(codes.FailedPrecondition, "Unexpected message type: %T", t)
		}
	}

	repo := svc.Repository{
		Id:   repoContext.GetCreatorId(),
		Name: repoContext.GetName(),
		Url:  fmt.Sprintf("https://gi.example.com/%s/%s", repoContext.CreatorId, repoContext.GetName()),
	}

	r := svc.RepoCreateReply{
		Repo: &repo,
		Size: int64(len(data)),
	}

	log.Println("Client disconnected")
	return stream.SendAndClose(&r)
}

func registerServer(s *grpc.Server) {
	svc.RegisterRepoServer(s, &repoService{})
	reflection.Register(s)
}

func startServer(s *grpc.Server, l net.Listener) error {
	return s.Serve(l)
}
