package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/kamenlilovgocourse/gocourse/project/cachegrpc"
	"github.com/kamenlilovgocourse/gocourse/project/item"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	serverAddr = flag.String("addr", "localhost:3030", "The server address in the format of host:port")
)

// Parse a command input via bufio.NewReader.ReadString, truncate any trailing cr and lf,
// and return the first word (the command name) and the second part (the command string)
func parseCommand(input string) (iCmd, iParam string) {
	input = strings.TrimSuffix(input, "\n")
	input = strings.TrimSuffix(input, "\r")
	inputPIndex := strings.Index(input, " ")
	iParam = ""
	iCmd = input
	if inputPIndex >= 0 {
		origInputPIndex := inputPIndex
		for {
			inputPIndex++
			if inputPIndex >= len(input) || input[inputPIndex] != ' ' {
				break
			}
		}
		iParam = input[inputPIndex:]
		iCmd = input[:origInputPIndex]
	}
	iCmd = strings.ToLower(iCmd)
	return iCmd, iParam
}

// Handle the 'subscribe' user command. This function is run in a separate goroutine.
// It will issue a SubscribeItem gRPC call passing the server an item.ID obtained
// from the console, and will repeatedly listen on the formed stream and print
// any received subscriptions on the console
func subscribeListener(client cachegrpc.CacheServerClient, id item.ID) {
	ip := cachegrpc.GetItemParams{Owner: id.Owner, Service: id.Service, Name: id.Name}
	stream, err1 := client.SubscribeItem(context.Background(), &ip)
	if err1 != nil {
		fmt.Printf("Error subscribing to %s: %v\n", id.Compose(), err1)
		return
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Printf("Error while receiving subscription: %v\n", err)
			return
		}
		fmt.Printf("Received sub for %s: new value %s\n", id.Compose(), res.Value)
	}
}

// Display help on the available commands for the command line client
func commandHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("set user:service:item=value,expiry sets an item in the cache")
	fmt.Println("get user:service:item retrieves an item from the cache")
	fmt.Println("subscribe user:service:item subscribes for updates to a shared cached item")
	fmt.Println("quit quits the client")
}

// Main client routine
func main() {
	flag.Parse()
	fmt.Printf("cacheclient seeking server at %s\n", *serverAddr)

	// Contact the server
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := cachegrpc.NewCacheServerClient(conn)

	// Issue a GetClientID call and just display the received value on the
	// console. The user is not obligated to use this value as the owner name
	// in set, get and subscribe calls, but it's a good practice to keep your
	// own private ID in a multiuser environment
	ctx := context.Background()
	clientID, err := client.GetClientID(ctx, &cachegrpc.AssignClientID{})
	if err != nil {
		log.Fatalf("client.GetClientID failed: %v", err)
	}
	fmt.Printf("Server assigned us client id %s\n", clientID.Id)

	linereader := bufio.NewReader(os.Stdin)
	commandHelp()
	for {
		// ReadString will block until the delimiter is entered
		input, err := linereader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("An error occured while reading input ", err)
			}
			return
		}

		iCmd, iParam := parseCommand(input)
		switch {
		case iCmd == "set":
			// The set command accepts an assignment as its parameter. Parse it out, then call
			// the server to set the data item as requested by the client
			iassn, err := item.ParseAssignment(iParam)
			if err != nil {
				fmt.Println("Error in expression: ", err)
				continue
			}
			ip := cachegrpc.SetItemParams{Owner: iassn.Id.Owner, Service: iassn.Id.Service, Name: iassn.Id.Name, Value: iassn.Value}
			if iassn.Expiry == nil {
				ip.Expiry = nil
			} else {
				ip.Expiry = timestamppb.New(*iassn.Expiry)
			}
			client.SetItem(ctx, &ip)

		case iCmd == "get":
			// The get command accepts an item ID as its parameter. Parse it out, then
			// call the server to retrieve and print the value, if any
			iassn := item.ID{}
			err := iassn.Parse(iParam)
			if err != nil {
				fmt.Println("Error in expression: ", err)
				continue
			}
			ip := cachegrpc.GetItemParams{Owner: iassn.Owner, Service: iassn.Service, Name: iassn.Name}
			ipres, err2 := client.GetItem(ctx, &ip)
			if err2 != nil {
				fmt.Println("Error from service: ", err2)
				continue
			}
			fmt.Printf("Result: %s\n", ipres.Value)

		case iCmd == "subscribe":
			// The subscribe command accepts an item ID as its parameter. Parse it out, then
			// spawn a goroutine to perform asynchronous listening to the formed stream request
			iassn := item.ID{}
			err := iassn.Parse(iParam)
			if err != nil {
				fmt.Println("Error in expression: ", err)
				continue
			}
			go subscribeListener(client, iassn)

		case iCmd == "quit":
			// quit quits the application as an alternative to ctrl+C
			var t cachegrpc.AssignClientID
			t.Dummy = 1
			return
		default:
			fmt.Printf("Unrecognized command %s with parameter %s\n", iCmd, iParam)
			commandHelp()
		}

	}
}
