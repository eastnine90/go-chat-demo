package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	pb "github.com/eastnine90/go-chat-demo/protos"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	defaultName = "anon"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "chat user's name")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewChatClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//check if server is up
	_, err = client.EnterOrCreate(ctx, &pb.ChatRoomRequest{})
	if err != nil {
		log.Fatalln(err)
	}

	id, receiveStream := promptAndEnter(ctx, client)
	for receiveStream == nil {
		id, receiveStream = promptAndEnter(ctx, client)
	}
	sendStream, _ := client.SendChat(ctx)

	go send(id, sendStream)
	receive(receiveStream)

}

func promptAndEnter(ctx context.Context, client pb.ChatClient) (string, pb.Chat_EnterOrCreateClient) {
	commandline := prompt.Input(">> ", func(d prompt.Document) []prompt.Suggest {
		s := []prompt.Suggest{
			{Text: "list", Description: "List names of existing chatrooms"},
			{Text: "create", Description: "create new chat room, enter if it already exists"},
			{Text: "enter", Description: "enter existing chatroom, equivalent to create"},
		}

		return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
	})
	commands := strings.Split(commandline, " ")

	if len(commands) == 0 || len(commands) > 2 {
		fmt.Println("invalid command")
		return "", nil
	}

	switch commands[0] {
	case "list":
		list, _ := client.GetList(ctx, &emptypb.Empty{})
		if list != nil {
			for roomEntry := range list.ChatRooms {
				fmt.Println(roomEntry)
			}
		}
		return "", nil
	case "create", "enter":
		id := uuid.New().String()
		stream, _ := client.EnterOrCreate(ctx, &pb.ChatRoomRequest{ChatRoomName: commands[1], UserId: id})
		return id, stream
	default:
		return "", nil
	}
}

func send(id string, stream pb.Chat_SendChatClient) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if err := stream.Send(&pb.ChatMessage{Uuid: id, Name: *name, Message: text}); err != nil {
			log.Fatalf("failed to send message: %v", err)
			return
		}
	}
}

func receive(stream pb.Chat_EnterOrCreateClient) {
	for {
		res, err := stream.Recv()
		if res != nil {
			fmt.Print(res.ChatMessage.Name + ": " + res.ChatMessage.Message)
		}

		if err != nil {
			break
		}
	}
}
