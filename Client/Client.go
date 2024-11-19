package main

import (
	cc "Server"
	"context"
	"flag"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var CurrentHighestBid int64

func ServerConnection(client cc.ServerClient){
	
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
