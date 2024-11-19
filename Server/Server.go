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
	"google.golang.org/grpc/credentials/insecure"
)

var leader bool
var ip string = "localhost:"
var otherServerPort string

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
	if leader {
		if s.AuctionClosed {
			return &cc.Acknowladgement{Ack: "fail"}, nil
		}

		if s.HighestBid < Amount.Value {
			s.HighestBid = Amount.Value
			s.Bidder = Amount.Id

			if otherServerPort != "" {
				conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
				newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

				client := cc.NewServerClient(conn)
				client.LeaderToFollowerUpdate(newContext, Amount)
			}

			return &cc.Acknowladgement{Ack: "success"}, nil
		}

		return &cc.Acknowladgement{Ack: "fail"}, nil
	}
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
	var listenPort string
	flag.StringVar(&listenPort, "lp", "5050", "Sets the listenPort of the node")

	flag.StringVar(&otherServerPort, "osp", "5051", "Sets the port of the other node")

	// var die bool
	// flag.BoolVar(&die, "d", false, "determines what server can die")

	flag.BoolVar(&leader, "l", false, "Specifies if this server is the leader")

	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("%s%s", ip, listenPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Printf("Now listening to listenPort: %s", listenPort)
	}

	grpcServer := grpc.NewServer()
	var s *server = NewServer()
	cc.RegisterServerServer(grpcServer, s)

	go s.AuctionTimer()
	grpcServer.Serve(lis)

	// if die {
	// 	go s.kill()
	// }

	// <-deathChan

	//Nothing after this runs
}
