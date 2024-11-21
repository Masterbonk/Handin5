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

var leader bool
var ip string = "localhost:"
var otherServerPort string

// Unix timestamp of when the auction started
var StartTime int64

var auctionDuration int64

type server struct {
	cc.UnimplementedServerServer
	HighestBid    int64
	AuctionClosed bool
	Bidder        int32
}

func (s *server) Result(ctx context.Context, empty *cc.Empty) (*cc.Outcome, error) {
	if leader {
		fmt.Printf("Is leader - Return result!\n")
		return &cc.Outcome{
			WinnerId:     s.Bidder,
			HighestValue: s.HighestBid,
			AuctionDone:  s.AuctionClosed,
		}, nil
	} else {
		fmt.Printf("Not leader - Get leader result!\n")

	}
}

func (s *server) Bid(ctx context.Context, amount *cc.Amount) (*cc.Acknowladgement, error) {
	fmt.Printf("Received amount!\n")

	if leader {

	}

	if s.AuctionClosed || amount.Value <= s.HighestBid {
		if s.AuctionClosed {
			fmt.Printf("Auction is closed - return fail\n")
		} else {
			fmt.Printf("Bid was too low - return fail\n")
		}
		return &cc.Acknowladgement{Ack: "fail"}, nil
	}

	s.HighestBid = amount.Value
	s.Bidder = amount.Id
	fmt.Printf("Bid was successful - return success\n")
	return &cc.Acknowladgement{Ack: "success"}, nil
}

func NewServer() *server {
	s := &server{HighestBid: 0, AuctionClosed: false}
	return s
}

func main() {

	auctionDuration = 5

	//Making server
	var listenPort string
	flag.StringVar(&listenPort, "lp", "5050", "Sets the listenPort of the node")
	flag.StringVar(&otherServerPort, "osp", "5051", "Sets the port of the other node")
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

	StartTime = time.Now().Unix()

	s.AuctionClosed = false
	s.Bidder = -1
	s.HighestBid = 0

	grpcServer.Serve(lis)
}
