# Handin5

## To run the servers
To run the server, it is recommended that you start 4 terminals up.

The first two terminals should be servers, the commands assumes that you are at the top of the folder for the program, so one layer abover Server.go. Servers can be started with the base command:
`go run Server/Server.go` 

Servers have 3 different flags assosiated with the command (flags are stuff like the -p in `go run prog.go -p 5`).

- The first flag is `-l` which contains a true or fase value, that determines whether this server is the starting leader of the system. Default value is false. An example of a terminal command using this flag is: <br/>
`go run Server/Server.go -l true`
- The second flag is `-lp` which determines Listening Port, ie. the port other clients and servers will have to refrence. It's base value is "5050". An example of a terminal command using this flag is: <br/>
`go run Server/Server.go -lp "5051"`
- The third flag is `-osp` which stands for Other Servers Port. This tells the server what the other servers listening port is, so it can send messages to it. It's default value is "5051". An example of a terminal command using this flag is: <br/>
`go run Server/Server.go -lp "5051" -osp "5050"`

It is recommended to start the leader server with this command: <br/>
`go run Server/Server.go -l true`

And the follower server with this command: <br/>
`go run Server/Server.go -lp "5051" -osp "5050"`

## To run the clients
The other two terminals should be the clients. It is recommended to make two clients where one targets the follower server and the other targets the leader server. This will allow the program to more easily showcase it's failure-stop resilience.

Clients can take 3 different flags:
- The first flag is `-i` which determines the ID of the client. This value must be higher than 0 and must be given or the program will not run. 

- The second flag is `-p1` which is the first server port it targets. Should this server crash it will move onto the server at `-p2`.

- The third flag is `-p2` which is the second server port the client will target if the first crashes. 

An example of a client being made using these flags is: <br/>
`go run Client/Client.go -i 1 -p1 "5050" -p2 "5051"`

It is recommended to create the first client with this command: <br/>
`go run Client/Client.go -i 1 -p1 "5050" -p2 "5051"`

And the second with this command: <br/>
`go run Client/Client.go -i 2 -p1 "5051" -p2 "5050"`