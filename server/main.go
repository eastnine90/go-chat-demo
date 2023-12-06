package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/eastnine90/go-chat-demo/protos"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	pb.UnimplementedChatServer
	chatRooms    map[string]*pb.ChatRoomsResponse_ChatRoom
	mu           sync.Mutex
	userChatRoom map[string]*pb.ChatRoomsResponse_ChatRoom
	streams      map[string]*pb.Chat_EnterOrCreateServer
}

func Server() *server {
	return &server{
		chatRooms:    make(map[string]*pb.ChatRoomsResponse_ChatRoom),
		userChatRoom: make(map[string]*pb.ChatRoomsResponse_ChatRoom),
		streams:      make(map[string]*pb.Chat_EnterOrCreateServer),
	}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	s := Server()
	pb.RegisterChatServer(srv, s)

	srv.Serve(lis)
	srv.GracefulStop()
}

// implementation of server-side GetList
func (s *server) GetList(ctx context.Context, empty *empty.Empty) (*pb.ChatRoomsResponse, error) {
	return &pb.ChatRoomsResponse{ChatRooms: s.chatRooms}, nil
}

// implementation of server-side EnterOrCreate
func (s *server) EnterOrCreate(request *pb.ChatRoomRequest, stream pb.Chat_EnterOrCreateServer) error {
	uuid := request.GetUserId()
	name := request.GetChatRoomName()
	_, hasId := s.streams[uuid]
	if hasId {
		return errors.New("key collision: uuid already exists")
	}

	s.mu.Lock()
	chatRoom, exists := s.chatRooms[name]
	if !exists {
		chatRoom = &pb.ChatRoomsResponse_ChatRoom{Name: name, Users: make(map[string]bool)}
		s.chatRooms[name] = chatRoom
	}
	chatRoom.Users[uuid] = true
	s.userChatRoom[uuid] = chatRoom
	s.streams[uuid] = &stream
	s.mu.Unlock()

	defer s.exitRoom(chatRoom, uuid)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		default:
		}
	}
}

func (s *server) SendChat(stream pb.Chat_SendChatServer) error {

	for {
		chatMessage, err := stream.Recv()
		if err != nil {
			return err
		}
		uuid := chatMessage.GetUuid()
		chatRoom := s.userChatRoom[uuid]
		for subscriber := range chatRoom.Users {
			if subscriber != uuid {
				(*s.streams[subscriber]).Send(&pb.ChatStreamResponse{
					Timestamp:   timestamppb.Now(),
					ChatMessage: chatMessage,
				})
			}
		}
	}
}

func (s *server) exitRoom(chatRoom *pb.ChatRoomsResponse_ChatRoom, uuid string) {
	s.mu.Lock()
	delete(chatRoom.Users, uuid)
	delete(s.userChatRoom, uuid)
	if len(chatRoom.Users) == 0 {
		delete(s.chatRooms, chatRoom.Name)
	}
	s.mu.Unlock()
}

// implementation of Stream RPC
// func (s *server) Stream(stream pb.ChatRoom_StreamServer) error {
// 	uuid := uuid.New().String()
// 	defer delete(s.Clients, uuid)
// 	s.Clients[uuid] = &stream
// 	for {
// 		req, err := stream.Recv()
// 		if err == io.EOF {
// 			continue
// 		} else if err != nil {
// 			return err
// 		}

// 		for id, client := range s.Clients {
// 			if uuid != id {
// 				(*client).Send(&pb.StreamResponse{
// 					Timestamp: timestamppb.Now(),
// 					ClientMessage: &pb.StreamResponse_Message{
// 						Name:    req.Name,
// 						Message: req.Message}})
// 			}
// 		}
// 	}
// }
