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

var waitForOtherMillis int64

// Unix timestamp of when the auction started
var StartTime int64
var auctionDuration int64
var otherServerClient cc.ServerClient

type server struct {
	cc.UnimplementedServerServer
	HighestBid int64
	Bidder     int32
}

func (s *server) Result(ctx context.Context, empty *cc.Empty) (*cc.Outcome, error) {
	if leader {
		fmt.Printf("Is leader - Return result!\n")
		fmt.Println("")
		return &cc.Outcome{
			WinnerId:     s.Bidder,
			HighestValue: s.HighestBid,
			AuctionDone:  s.GetAuctionDone(),
		}, nil
	} else {
		fmt.Printf("Not leader - Get leader result!\n")
		var outcomeChan chan cc.Outcome = make(chan cc.Outcome)
		var outcome cc.Outcome
		var receivedOutcome bool

		go getResultFromLeader(outcomeChan)
		var timeout = time.After(time.Duration(waitForOtherMillis) * time.Millisecond)

		select {
		case outcome = <-outcomeChan:
			receivedOutcome = true
		case <-timeout:
			receivedOutcome = false
		}

		if receivedOutcome {
			fmt.Printf("Received outcome from leader in time!\n")
			fmt.Println("")
			return &outcome, nil
		} else {
			fmt.Printf("Did not receive outcome from leader in time - becoming leader!\n")
			leader = true
			fmt.Println("")
			return &cc.Outcome{
				WinnerId:     s.Bidder,
				HighestValue: s.HighestBid,
				AuctionDone:  s.GetAuctionDone(),
			}, nil
		}
	}
}

func getResultFromLeader(outChan chan cc.Outcome) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	out, err := otherServerClient.Result(newContext, &cc.Empty{})
	if err == nil {
		outChan <- *out
	}
}

func (s *server) Bid(ctx context.Context, amount *cc.Amount) (*cc.Acknowladgement, error) {
	fmt.Printf("Received bid!\n")

	if leader {
		return s.LeaderHandleBid(amount)
	} else {
		return s.FollowerHandleBid(amount)
	}
}

func (s *server) LeaderHandleBid(amount *cc.Amount) (*cc.Acknowladgement, error) {
	fmt.Printf("Handle bid as leader\n")

	// check validity
	auctionClosed := s.GetAuctionDone()

	if auctionClosed || amount.Value <= s.HighestBid {
		if auctionClosed {
			fmt.Printf("Auction is closed - return fail\n")
		} else {
			fmt.Printf("Bid (value %d) was too low - return fail\n", amount.Value)
		}
		return &cc.Acknowladgement{Ack: "fail"}, nil
	}

	// update follower
	var timeout = time.After(time.Duration(waitForOtherMillis) * time.Millisecond)
	var channel chan bool = make(chan bool)
	go SendUpdateToFollower(channel, amount)

	// wait for the update to be confirmed or a timeout
	select {
	case <-channel:
	case <-timeout:
	}

	// update own state
	s.HighestBid = amount.Value
	s.Bidder = amount.Id

	fmt.Printf("Bid (value %d) was successful - return success\n", amount.Value)
	return &cc.Acknowladgement{Ack: "success"}, nil
}

func SendUpdateToFollower(channel chan bool, amount *cc.Amount) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	otherServerClient.LeaderToFollowerUpdate(newContext, &cc.ServerToServer{
		Value: amount.Value,
		Id:    amount.Id})

	channel <- true
}

// Called on the follower by the leader
func (s *server) LeaderToFollowerUpdate(ctx context.Context, message *cc.ServerToServer) (*cc.Empty, error) {
	fmt.Printf("Received update from leader!\n")
	if leader {
		panic("Leader received update from other leader!")
	}

	var msg cc.ServerToServer = *message
	if msg.Value <= s.HighestBid {
		panic("Received update with bid less than or equal to current highest bid")
	}

	s.HighestBid = msg.Value
	s.Bidder = msg.Id

	return &cc.Empty{}, nil
}

func (s *server) FollowerHandleBid(amount *cc.Amount) (*cc.Acknowladgement, error) {

	fmt.Printf("Handle bid as follower!\n")

	// send bid to leader
	var timeout = time.After(time.Duration(waitForOtherMillis) * time.Millisecond)
	var channel chan *cc.Acknowladgement = make(chan *cc.Acknowladgement)
	var ack *cc.Acknowladgement
	go SendBidToLeader(amount, channel)

	var receivedAck bool

	select {
	case ack = <-channel:
		receivedAck = true
	case <-timeout:
		receivedAck = false
	}

	if receivedAck {
		fmt.Printf("Received ack from leader - returning it to client\n")
		return ack, nil
	}

	// if follower did not receive acknowledgement from leader in time, become leader and handle bid
	leader = true
	fmt.Printf("Did not receive ack from leader in time - becoming leader\n")
	return s.LeaderHandleBid(amount)
}

func SendBidToLeader(amount *cc.Amount, channel chan *cc.Acknowladgement) {
	fmt.Printf("Forwarding bid to leader!!\n")
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	ack, _ := otherServerClient.Bid(newContext, amount)
	channel <- ack
}

func (s *server) GetAuctionDone() bool {
	elapsedTime := time.Now().Unix() - StartTime
	return elapsedTime >= auctionDuration
}

func NewServer() *server {
	s := &server{HighestBid: 0, Bidder: -1}
	return s
}

func main() {

	auctionDuration = 10
	waitForOtherMillis = 2000

	//Making server
	var listenPort string
	flag.StringVar(&listenPort, "lp", "5050", "Sets the listenPort of the node")
	flag.StringVar(&otherServerPort, "osp", "5051", "Sets the port of the other node")
	flag.BoolVar(&leader, "l", false, "Specifies if this server is the leader")
	flag.Parse()

	if leader {
		fmt.Printf("Starting as leader!\n")
	} else {
		fmt.Printf("Starting as follower!\n")
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s%s", ip, listenPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	} else {
		log.Printf("Now listening to listenPort: %s", listenPort)
	}

	grpcServer := grpc.NewServer()
	var s *server = NewServer()
	cc.RegisterServerServer(grpcServer, s)

	// make client to other server
	conn, err := grpc.NewClient(ip+otherServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3\n")
	}
	otherServerClient = cc.NewServerClient(conn)

	// start auction
	StartTime = time.Now().Unix()

	s.Bidder = -1
	s.HighestBid = -1

	if leader {
		s.LeaderHandleBid(&cc.Amount{
			Value: 0,
			Id:    -1})
	}

	grpcServer.Serve(lis)
}
