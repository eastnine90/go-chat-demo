package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	pb "github.com/eastnine90/go-chat-demo/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	c := pb.NewChatRoomClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := c.Stream(ctx)

	if err != nil {
		log.Fatalln(err)
	}

	go send(s)
	receive(s)

}

func send(client pb.ChatRoom_StreamClient) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if err := client.Send(&pb.StreamRequest{Name: *name, Message: text}); err != nil {
			log.Fatalf("failed to send message: %v", err)
			return
		}
	}
}

func receive(client pb.ChatRoom_StreamClient) {
	for {
		res, _ := client.Recv()
		fmt.Print(res.ClientMessage.Name + ": " + res.ClientMessage.Message)
	}
}
