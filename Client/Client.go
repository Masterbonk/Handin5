package main

import (
	cc "Server"
	"flag"
	"log"
	"math/rand/v2"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CurrentHighestBid int64

func ServerConnection(client cc.ServerClient) {

}

func main() {
	CurrentHighestBid = 0

	ip := "localhost:"

	var port1 string
	flag.StringVar(&port1, "p1", "5050", "Sets the port of the server 1")

	var port2 string
	flag.StringVar(&port2, "p2", "5051", "Sets the port of the server 2")
	flag.Parse()

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

	var auctionClosed bool = false
	for auctionClosed == false {
		var outcome cc.Outcome = cc.Outcome{
			AuctionDone: false,
			HighestValue: -1,
    		WinnerId:-1}
		go getResult(&outcome, port)
		time.Sleep(500 * time.Millisecond)

		if outcome.WinnerId != -1 {
			if !outcome.AuctionDone {
				if id != outcome.WinnerId {
					current = CurrentHighestBid
					betValue = CurrentHighestBid+rand.Int64N(20)+1
					go bid(betValue, port)
					time.Sleep(500 * time.Millisecond)
					if current == CurrentHighestBid {
						go bid(betValue, port)
					}
				}
			}
		}
	}

	/*
		students, err := client1.GetStudents(context.Background(), &cc.Empty{})
		if err != nil {
			log.Fatalf("Not working 4")
		}

		for _, students := range students.Students {
			println(" - " + students)
		}
	*/
}
