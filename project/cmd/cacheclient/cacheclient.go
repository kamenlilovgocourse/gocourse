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

func commandHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("set user:service:item=value,expiry sets an item in the cache")
	fmt.Println("get user:service:item retrieves an item from the cache")
	fmt.Println("quit quits the client")
}

func main() {
	flag.Parse()
	fmt.Printf("cacheclient seeking server at %s\n", *serverAddr)

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := cachegrpc.NewCacheServerClient(conn)

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
			fmt.Printf("Result: %s, %#v", ipres.Value, ipres.Expiry)

		case iCmd == "quit":
			var t cachegrpc.AssignClientID
			t.Dummy = 1
			return
		default:
			fmt.Printf("Unrecognized command %s with parameter %s\n", iCmd, iParam)
			commandHelp()
		}

	}
}
