package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	// cachegrpc "github.com/kamenlilovgocourse/gocourse/project/cachegrpc"
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
	fmt.Println("Available commands:")
	fmt.Println("set user:service:item=value,expiry sets an item in the cache")
	fmt.Println("quit quits the client")
}

func main() {
	fmt.Printf("cacheclient\n")
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
			fmt.Printf("input=%s par=%s\n", iCmd, iParam)
		case iCmd == "quit":
			//var t cachegrpc.AssignClientID
			//t.Dummy = 1
			return
		default:
			fmt.Printf("Unrecognized command %s with parameter %s\n", iCmd, iParam)
			commandHelp()
		}

	}
}
