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
	} else {
		var temp cc.Acknowladgement = cc.Acknowladgement{
			Ack: "exception"}
		go betToLeader(Amount, &temp)
		time.Sleep(500 * time.Millisecond)
		if temp.Ack != "exception" {
			return &temp, nil
		} else {
			leader = true
			otherServerPort = ""
			newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
			return s.bid(newContext, Amount)
		}
	}

}

func betToLeader(Amount *cc.Amount, temp *cc.Acknowladgement) {
	conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))

	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	client := cc.NewServerClient(conn)
	temp, _ = client.Bid(newContext, Amount)

}

func resultFromLeader(temp *cc.Outcome) {

	conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))

	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	client := cc.NewServerClient(conn)
	temp, _ = client.Result(newContext, &cc.Empty{})

}

func (s *server) result(ctx context.Context, Empty *cc.Empty) (*cc.Outcome, error) {
	if leader {
		return &cc.Outcome{
			AuctionDone:  s.AuctionClosed,
			HighestValue: s.HighestBid,
			WinnerId:     s.Bidder}, nil
	} else {

		var temp cc.Outcome = cc.Outcome{
			AuctionDone:  false,
			HighestValue: -1,
			WinnerId:     -1}

		go resultFromLeader(&temp)
		time.Sleep(500 * time.Millisecond)
		if temp.WinnerId != -1 {
			return &temp, nil
		} else {
			leader = true
			otherServerPort = ""
			newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
			return s.result(newContext, &cc.Empty{})
		}
	}

}

func main() {

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

	//Nothing after this runs
}
