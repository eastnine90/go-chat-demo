package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	pb "github.com/eastnine90/go-chat-demo/protos"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedChatRoomServer
	Clients map[string]*pb.ChatRoom_StreamServer
}

func Server() *server {
	return &server{Clients: make(map[string]*pb.ChatRoom_StreamServer)}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	s := Server()
	pb.RegisterChatRoomServer(srv, s)

	srv.Serve(lis)
	srv.GracefulStop()
}

// implementation of Stream RPC
func (s *server) Stream(stream pb.ChatRoom_StreamServer) error {
	uuid := uuid.New().String()
	defer delete(s.Clients, uuid)
	s.Clients[uuid] = &stream
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			continue
		} else if err != nil {
			return err
		}

		for id, client := range s.Clients {
			if uuid != id {
				(*client).Send(&pb.StreamResponse{
					Timestamp: timestamppb.Now(),
					ClientMessage: &pb.StreamResponse_Message{
						Name:    req.Name,
						Message: req.Message}})
			}
		}
	}
}
