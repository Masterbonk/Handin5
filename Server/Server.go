package main

import (
	cc "Server"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
)

type server struct {
	cc.UnimplementedServerServer
	HighestBid    int64
	AuctionClosed bool
}

func NewServer() *server {
	s := &server{HighestBid: 0, AuctionClosed: false}
	return s
}

func (s *server) AuctionTimer() {
	time.Sleep(10*time.Second)
	s.AuctionClosed = true
}

func (s *server) bid(ctx context.Context, Amount *cc.Amount) (*cc.Acknowladgement, error) {
	if s.AuctionClosed {
		return &cc.Acknowladgement{
			Ack: "Failure"}, nil
	}

	if s.HighestBid < Amount.Value {
		s.HighestBid = Amount.Value
	}
}

func (s *server) result(ctx context.Context, Empty *cc.Empty) (*cc.Outcome, error) {
	return &cc.Outcome{
		AuctionDone: s.AuctionClosed,
		HighestValue: s.HighestBid}, nil
}


func main() {

	//Making server
	ip := "localhost"

	var port string
	flag.StringVar(&port, "p", "5050", "Sets the port of the node")

	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Printf("Now listening to port: %s", port)
	}

	grpcServer := grpc.NewServer()

	var s *server = NewServer()

	cc.RegisterServerServer(grpcServer, s)

	go s.AuctionTimer()
	grpcServer.Serve(lis)
	//Nothing after this runs
}
