package main

import (
	cc "Client"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ip string
var targetPort string

var queue []string
var queueLock sync.Mutex

var lastMsg string

type server struct {
	cc.UnimplementedClientServer
	Msg string
	Key int32
}

func NewServer(startNode bool) *server {
	var s *server
	if startNode {
		s = &server{Msg: "", Key: 737}
	} else {
		s = &server{Msg: "", Key: 0}
	}
	return s
}

func (s *server) PassAlong(ctx context.Context, clientMessage *cc.ClientMessage) (*cc.Empty, error) {
	s.Key = clientMessage.Key

	// alter message
	queueLock.Lock()
	for _, scripture := range queue {
		clientMessage.Msg = fmt.Sprintf("%s%s", clientMessage.Msg, scripture)
	}
	queue = nil
	queueLock.Unlock()

	if clientMessage.Msg != lastMsg {
		fmt.Println(clientMessage.Msg)
	}
	lastMsg = clientMessage.Msg

	//Making client
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	var serverAddress = fmt.Sprintf("%s:%s", ip, targetPort)
	conn, err := grpc.NewClient(serverAddress, opts...)

	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}

	// create client
	client := cc.NewClientClient(conn)

	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	go func() {
		// close connection when function terminates
		defer conn.Close()
		client.PassAlong(newContext, clientMessage)
	}()

	s.Key = 0
	return &cc.Empty{}, nil
}

func main() {

	//Making server
	ip := "localhost"

	var port string
	flag.StringVar(&port, "p", "5050", "Sets the port of the node")

	flag.StringVar(&targetPort, "tp", "5051", "Sets the port of the target node")

	var startNode bool
	flag.BoolVar(&startNode, "s", false, "Determines if the node should start, once it's set up")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Printf("Now listening to port: %s", port)
	}

	grpcServer := grpc.NewServer()

	var s *server = NewServer(startNode)

	cc.RegisterClientServer(grpcServer, s)

	if startNode {
		newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
		go s.PassAlong(newContext, &cc.ClientMessage{Key: s.Key, Msg: ""})
	}

	go QueueUp(port)
	grpcServer.Serve(lis)

}

func QueueUp(port string) {
	for {
		// range is [3:4]
		var delay int = 5 + rand.IntN(5)
		time.Sleep(time.Duration(delay) * time.Second)

		queueLock.Lock()
		words := []string{"hello", "hi", "hello world", "no u"}
		word := words[rand.IntN(len(words))]
		msg := fmt.Sprintf("%s: %s\n", port, word)

		queue = append(queue, msg)
		queueLock.Unlock()
	}
}
