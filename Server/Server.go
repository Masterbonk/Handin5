package main

import (
	cc "Server"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"time"

	"google.golang.org/grpc"
)

var deathChan chan bool

type server struct {
	cc.UnimplementedServerServer
	HighestBid    int64
	AuctionClosed bool
	Bidder        int32
}

func NewServer() *server {
	s := &server{HighestBid: 0, AuctionClosed: false}
	return s
}

func (s *server) AuctionTimer() {
	time.Sleep(10 * time.Second)
	s.AuctionClosed = true
}

func (s *server) bid(ctx context.Context, Amount *cc.Amount) (*cc.Acknowladgement, error) {
	if s.AuctionClosed {
		return &cc.Acknowladgement{
			Ack: "Failure"}, nil
	}

	if s.HighestBid < Amount.Value {
		s.HighestBid = Amount.Value
		s.Bidder = Amount.Id

		return &cc.Acknowladgement{
			Ack: "Success"}, nil

	}

	return &cc.Acknowladgement{
		Ack: "Failure"}, nil

}

func (s *server) result(ctx context.Context, Empty *cc.Empty) (*cc.Outcome, error) {
	return &cc.Outcome{
		AuctionDone:  s.AuctionClosed,
		HighestValue: s.HighestBid,
		WinnerId:     s.Bidder}, nil
}

func (s *server) kill() {
	time.Sleep(time.Duration(rand.IntN(12)) * time.Second)
	deathChan <- true

}

func main() {

	deathChan = make(chan bool)

	//Making server
	ip := "localhost"

	var port string
	flag.StringVar(&port, "p", "5050", "Sets the port of the node")

	var die bool
	flag.BoolVar(&die, "d", false, "determines what server can die")

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
	go grpcServer.Serve(lis)

	if die {
		go s.kill()
	}

	<-deathChan

	//Nothing after this runs
}
