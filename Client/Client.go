package main

import (
	cc "Server"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CurrentHighestBid int64

var lock sync.Mutex

func getResult(outcome *cc.Outcome, client cc.ServerClient) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	var out *cc.Outcome
	var c cc.ServerClient = client
	out, _ = c.Result(newContext, &cc.Empty{})

	outcome = out
}

func bid(bet int64, id int, bidFailed *bool, client cc.ServerClient) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	var ack *cc.Acknowladgement
	var c cc.ServerClient = client
	ack, _ = c.Bid(newContext, &cc.Amount{Value: bet, Id: int32(id)})

	if ack.Ack == "success" {
		lock.Lock()
		if bet > CurrentHighestBid {
			CurrentHighestBid = bet
		}
		lock.Unlock()
	} else if ack.Ack == "fail" {
		*bidFailed = true
	} else if ack.Ack == "exception" {
		fmt.Printf("Received exception from server\n")
	}
}

func main() {
	CurrentHighestBid = 0

	ip := "localhost:"

	var id int
	flag.IntVar(&id, "i", -1, "Sets the ID of the client - must be unique")

	var port1 string
	flag.StringVar(&port1, "p1", "5050", "Sets the port of the server 1")

	var port2 string
	flag.StringVar(&port2, "p2", "5051", "Sets the port of the server 2")
	flag.Parse()

	if id == -1 {
		panic("Client ID must be specified")
	}

	conn1, err := grpc.NewClient(ip+port1, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3")
	}

	conn2, err := grpc.NewClient(ip+port2, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3")
	}

	client1 := cc.NewServerClient(conn1)
	client2 := cc.NewServerClient(conn2)

	var currentClient cc.ServerClient = client1

	var auctionClosed bool = false
	for !auctionClosed {
		var outcome = cc.Outcome{
			AuctionDone:  false,
			HighestValue: -1,
			WinnerId:     -1}
		var outcomePointer = &outcome
		go getResult(outcomePointer, currentClient)
		time.Sleep(500 * time.Millisecond)

		if (*outcomePointer).WinnerId == -1 {
			currentClient = client2
			continue
		}

		if !(*outcomePointer).AuctionDone {
			if id != int((*outcomePointer).WinnerId) {
				var current = CurrentHighestBid
				var betValue = CurrentHighestBid + rand.Int64N(20) + 1
				var bidFailed = false
				go bid(betValue, id, &bidFailed, currentClient)
				time.Sleep(500 * time.Millisecond)

				if current == CurrentHighestBid && !bidFailed {
					currentClient = client2
					go bid(betValue, id, &bidFailed, currentClient)
				}
			}
		} else {
			auctionClosed = true
			fmt.Printf("Auction is closed - Winner is client %d with bid %d\n", (*outcomePointer).WinnerId, (*outcomePointer).HighestValue)
		}
	}
}
