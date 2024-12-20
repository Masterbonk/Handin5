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

var outcome cc.Outcome

func getResult(client string) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	conn, err := grpc.NewClient(client, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3")
	}

	c := cc.NewServerClient(conn)

	

	out, _ := c.Result(newContext, &cc.Empty{})
	
	if out == nil{
		out = &cc.Outcome{
			AuctionDone: false,
			HighestValue: -1,
			WinnerId: -1}
	}
	outcome = *out
	
	CurrentHighestBid = outcome.HighestValue
}

func bid(bet int64, id int, bidFailed *bool, client string) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)

	time.Sleep(1000 * time.Millisecond)

	conn, err := grpc.NewClient(client, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3")
	}

	c := cc.NewServerClient(conn)

	ack, _ := c.Bid(newContext, &cc.Amount{Value: bet, Id: int32(id)})

	
		

	if ack == nil{
	 	ack = &cc.Acknowladgement{
		Ack: "exception"}
	}

	if ack.Ack == "success" {
		lock.Lock()
		if bet > CurrentHighestBid {
			CurrentHighestBid = bet
		}
		lock.Unlock()
	} else if ack.Ack == "fail" {
		*bidFailed = true
	} else if ack.Ack == "exception" {
		*bidFailed = true
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
	client1 := ip+port1
	client2 := ip+port2

	var currentClient string = client1

	betValue := CurrentHighestBid + rand.Int64N(20) + 1
	var bidFailed = false

	go bid(betValue, id, &bidFailed, currentClient)
	time.Sleep(3 * time.Second)

	var auctionClosed bool = false
	for !auctionClosed {
		outcome = cc.Outcome{
			AuctionDone:  false,
			HighestValue: -1,
			WinnerId:     -1}
		//var outcomePointer = &outcome
		go getResult(currentClient)
		time.Sleep(3 * time.Second)
		fmt.Printf("Result gotten: Auction done %t, Highest value %d, Winner is %d\n", outcome.AuctionDone, outcome.HighestValue, outcome.WinnerId)

		if outcome.WinnerId == -1 {
			fmt.Printf("Server 1 said no\n")
			currentClient = client2
			continue
		}

		if !outcome.AuctionDone {
			if id != int(outcome.WinnerId) {
				var current = CurrentHighestBid
				var betValue = CurrentHighestBid + rand.Int64N(20) + 1
				var bidFailed = false
				fmt.Printf("Sending bid 1: %d\n", betValue)
				go bid(betValue, id, &bidFailed, currentClient)
				time.Sleep(500 * time.Millisecond)

				if current == CurrentHighestBid && bidFailed {
					currentClient = client2
					fmt.Printf("First bid fail, Sending bid 2: %d\n", betValue)
					go bid(betValue, id, &bidFailed, currentClient)
				}
			}
		} else {
			auctionClosed = true
			fmt.Printf("Auction is closed - Winner is client %d with bid %d\n", outcome.WinnerId, outcome.HighestValue)
		}
	}
}
