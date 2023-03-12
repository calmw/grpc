package main

import (
	"context"
	"fmt"
	svc "github.com/calmw/grpc-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"strings"
	"time"
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

type userService struct {
	svc.UnimplementedUsersServer // 对于grpc中任何服务实现都是强制性的
}

type repoService struct {
	svc.UnimplementedRepoServer
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
	u := svc.User{
		Id:        in.Id,
		FirstName: components[0],
		LastName:  components[1],
		Age:       36,
	}
	return &svc.UserGetReply{User: &u}, nil
}

func (s *repoService) GetRepos(in *svc.RepoGetRequest, stream svc.Repo_GetReposServer) error {
	log.Printf("Received request for repo with CreatorId: %s Id:%s\n", in.CreatorId, in.Id)
	repo := svc.Repository{
		Id:   in.Id,
		Name: "test repo",
		Url:  "https://git.example.com/test/repo",
		Owner: &svc.User{
			Id:        in.CreatorId,
			FirstName: "Jane",
			LastName:  "han",
			Age:       36,
		},
	}

	cnt := 1
	for {
		repo.Name = fmt.Sprintf("repo-%d", cnt)
		repo.Url = fmt.Sprintf("https://git.example.com/test/%s", repo.Url)
		r := svc.RepoGetReply{
			Repo: &repo,
		}
		err := stream.Send(&r)
		if err != nil {
			return err
		}

		if cnt >= 5 {
			break
		}
		time.Sleep(time.Second * 2)
		cnt++
	}

	return nil
}

func registerServer(s *grpc.Server) {
	svc.RegisterUsersServer(s, &userService{})
	svc.RegisterRepoServer(s, &repoService{})
	reflection.Register(s)
}

func startServer(s *grpc.Server, l net.Listener) error {
	return s.Serve(l)
}
