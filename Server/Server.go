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
var ip string
var otherServerPort string

var recievedFromLeader bool

// Unix timestamp of when the auction started
var StartTime int64


var CurrentTime int64

var baseAck cc.Acknowladgement
var leaderResult cc.Outcome


var auctionDuration int64

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
	var counter = 0
	for !recievedFromLeader {
		time.Sleep(500 * time.Millisecond)
		counter++
		if counter == 10 {
			recievedFromLeader = true
		}
	}
	fmt.Println("The auctiontimer has started")
	for {
		if CurrentTime < StartTime+auctionDuration {
			time.Sleep(time.Second)
			CurrentTime++
		} else {
			fmt.Println(CurrentTime)
			break
		}
	}
	fmt.Println("Auction closed")
	s.AuctionClosed = true
}

// Called on the follower by the leader. Updates the state of the follower to match that of the leader
func (s *server) LeaderToFollowerUpdate(ctx context.Context, input *cc.ServerToServer) (*cc.Empty, error) {
	if !recievedFromLeader {
		recievedFromLeader = true
	}

	if leader {
		panic("WE ARE THE LEADER, BUT RECIEVE A UPDATE FROM A LEADER")
	}

	s.HighestBid = input.Value
	s.Bidder = input.Id

	if input.Time != StartTime {
		difference := input.Time - StartTime
		StartTime = input.Time
		CurrentTime = CurrentTime + difference
	}

	return &cc.Empty{}, nil
}

func (s *server) Bid(ctx context.Context, Amount *cc.Amount) (*cc.Acknowladgement, error) {
	if leader {
		if s.AuctionClosed {
			fmt.Printf("Leader bid, auction closed\n")
			return &cc.Acknowladgement{Ack: "fail"}, nil
		}

		fmt.Printf("Leader Highest big 1 %d, bidder %d\n", s.HighestBid, s.Bidder)

		if s.HighestBid < Amount.Value {
			s.HighestBid = Amount.Value
			s.Bidder = Amount.Id

				fmt.Printf("Leader Highest big 2 %d, bidder %d\n", s.HighestBid, s.Bidder)


			if otherServerPort != "" {
				conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
				newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

				client := cc.NewServerClient(conn)

				client.LeaderToFollowerUpdate(newContext, &cc.ServerToServer{
					Value: Amount.Value,
					Id:    Amount.Id,
					Time:  StartTime})
			}
			fmt.Println("Sending suc ack")
			return &cc.Acknowladgement{Ack: "success"}, nil
		}

		return &cc.Acknowladgement{Ack: "fail"}, nil
	} else {
		baseAck = cc.Acknowladgement{
			Ack: "exception"}

		fmt.Printf("Follower Highest big 1 %d, bidder %d\n", s.HighestBid, s.Bidder)
		go betToLeader(Amount)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("Follower Highest big 2 %d, bidder %d\n", s.HighestBid, s.Bidder)
		if baseAck.Ack != "exception" {
			return &baseAck, nil
		} else {
			fmt.Println("Exception leader change")
			leader = true
			otherServerPort = ""
			newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
			return s.Bid(newContext, Amount)
		}
	}

}

func betToLeader(Amount *cc.Amount) {
	conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))

	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	client := cc.NewServerClient(conn)

	
	base, _ := client.Bid(newContext, Amount)

	if base == nil{
		base = &cc.Acknowladgement{
		Ack: "exception"}
	}

	baseAck = *base

	fmt.Printf("baseAck betToLeader %s\n", baseAck.Ack)
}

func resultFromLeader() {

	conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))

	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	client := cc.NewServerClient(conn)
	temp, _ := client.Result(newContext, &cc.Empty{})


	if temp == nil{
		temp = &cc.Outcome{
			AuctionDone: false,
			HighestValue: -1,
			WinnerId: -1}
	}
	
	leaderResult = *temp

	//fmt.Printf("Result gotten from leader: Auction done %t, Highest value %d, Winner is %d\n", leaderResult.AuctionDone, leaderResult.HighestValue, leaderResult.WinnerId)

}

func (s *server) Result(ctx context.Context, Empty *cc.Empty) (*cc.Outcome, error) {
	if leader {
		return &cc.Outcome{
			AuctionDone:  s.AuctionClosed,
			HighestValue: s.HighestBid,
			WinnerId:     s.Bidder}, nil
	} else {

		leaderResult = cc.Outcome{
			AuctionDone:  false,
			HighestValue: -1,
			WinnerId:     -1}

		go resultFromLeader()
		time.Sleep(500 * time.Millisecond)

		// if leader responded, returned modified value temp.
		// else, become leader and return own result
		if leaderResult.WinnerId != -1 {
			return &leaderResult, nil
		} else {
			fmt.Println("WinnerId -1 leader change")
			leader = true
			otherServerPort = ""
			newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
			return s.Result(newContext, &cc.Empty{})
		}
	}

}

func main() {

	ip = "localhost:"

	auctionDuration = 30

	recievedFromLeader = false

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

	StartTime = time.Now().Unix()

	CurrentTime = StartTime
	go s.AuctionTimer()

	// give follower the start time
	if leader {
	 	recievedFromLeader = true
	 	if otherServerPort != "" {
	 		conn, _ := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	 		newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	 		client := cc.NewServerClient(conn)

	 		client.LeaderToFollowerUpdate(newContext, &cc.ServerToServer{Time: StartTime, Id: -1, Value: 0})
	 	}
	}

	fmt.Println("Server, begins to run")
	grpcServer.Serve(lis)
	fmt.Println("Server, endline should not do")
	//Nothing after this runs
}
