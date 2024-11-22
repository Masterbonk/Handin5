package main

import (
	cc "Server"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var currentHighestBid int64
var waitForServerMillis int64 = 1000
var waitBetweenTriesMillis int64 = 1000

var id int

func main() {
	currentHighestBid = 0

	ip := "localhost:"

	flag.IntVar(&id, "i", -1, "Sets the ID of the client - must be unique")

	var port1 string
	flag.StringVar(&port1, "p1", "5050", "Sets the port of the server 1")

	var port2 string
	flag.StringVar(&port2, "p2", "5051", "Sets the port of the server 2")
	flag.Parse()

	if id == -1 {
		panic("Client ID must be specified\n")
	}

	conn1, err := grpc.NewClient(ip+port1, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3\n")
	}

	conn2, err := grpc.NewClient(ip+port2, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working 3\n")
	}

	currentClient := cc.NewServerClient(conn1)
	client2 := cc.NewServerClient(conn2)

	var auctionClosed = false

	for !auctionClosed {
		var outcome cc.Outcome
		var receivedOutcome bool

		outcome, receivedOutcome = getResult(currentClient)

		if !receivedOutcome {
			if currentClient != client2 {
				// try again with client 2
				currentClient = client2
				fmt.Printf("Switching to server 2\n")
				continue
			} else {
				log.Fatalf("*** Both servers seem to have crashed! ***\n")
			}
		}

		if outcome.AuctionDone {
			auctionClosed = true
			fmt.Printf("Auction is closed! Winner is user %d with a bid of %d!\n", outcome.WinnerId, outcome.HighestValue)
			return
		}

		if outcome.WinnerId == int32(id) {
			fmt.Printf("Still the winner!\n")
			time.Sleep(time.Duration(waitBetweenTriesMillis) * time.Millisecond)
			continue
		}

		// bid
		var currentHighestBid int64 = rand.Int64N(20) + 1
		ack, receivedAck := makeBid(currentClient, currentHighestBid)

		var success bool = receivedAck && ack.Ack == "success"

		fmt.Printf("Success: %t, receivedAck: %t\n", success, receivedAck)

		if !receivedAck && currentClient != client2 {
			currentClient = client2
			fmt.Printf("Switching to server 2\n")
			ack, receivedAck = makeBid(currentClient, currentHighestBid)
		}

		success = receivedAck && ack.Ack == "success"

		if success {
			fmt.Printf("Successfully made bid with value %d\n", currentHighestBid)
		} else {
			fmt.Printf("Fail or exception in making bid!\n")
		}
		time.Sleep(time.Duration(waitBetweenTriesMillis) * time.Millisecond)
	}
}

func getResult(client cc.ServerClient) (cc.Outcome, bool) {
	fmt.Printf("\nGet result\n")
	var outcomeChannel chan cc.Outcome = make(chan cc.Outcome)
	var outcome cc.Outcome

	go getResultFromServer(client, outcomeChannel)
	var timeout = time.After(time.Duration(waitForServerMillis) * time.Millisecond)
	var receivedOutcome bool

	select {
	case outcome = <-outcomeChannel:
		receivedOutcome = true
		fmt.Printf("Received outcome in time! Winner is %d\n", outcome.WinnerId)
	case <-timeout:
		receivedOutcome = false
		fmt.Printf("Did not receive outcome in time!\n")
	}

	return outcome, receivedOutcome
}

func getResultFromServer(client cc.ServerClient, outcomeChannel chan cc.Outcome) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	out, err := client.Result(newContext, &cc.Empty{})
	if err == nil {
		outcomeChannel <- *out
	} else {
		fmt.Println(err)
	}
}

func makeBid(client cc.ServerClient, bidValue int64) (cc.Acknowladgement, bool) {
	fmt.Printf("Make bid with value %d\n", bidValue)
	var ackChan chan cc.Acknowladgement = make(chan cc.Acknowladgement)
	var ack cc.Acknowladgement
	go sendBidToServer(client, bidValue, ackChan)

	var timeout = time.After(time.Duration(waitForServerMillis) * time.Millisecond)
	var receivedAck bool

	select {
	case ack = <-ackChan:
		receivedAck = true
	case <-timeout:
		receivedAck = false
	}

	return ack, receivedAck
}

func sendBidToServer(client cc.ServerClient, bidValue int64, ackChannel chan cc.Acknowladgement) {
	newContext, _ := context.WithTimeout(context.Background(), 2000*time.Second)
	ack, err := client.Bid(newContext, &cc.Amount{Value: bidValue, Id: int32(id)})
	if err == nil {
		ackChannel <- *ack
	}
}
