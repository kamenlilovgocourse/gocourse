package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/kamenlilovgocourse/gocourse/project/cachegrpc"
	"github.com/kamenlilovgocourse/gocourse/project/server"
)

var (
	port = flag.Int("port", 3030, "The server port")
)

// Main routine for the cache item server
func main() {
	flag.Parse()
	fmt.Printf("cacheserver running on port %d\n", *port)

	// Establish a listening port to be used with http2 and gRPC
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Spawn a goroutine to handle expiring values. We will notify it that
	// it's time to shut down by sending a struct{} on the NotifyServerThreadShutdown
	// channel
	server.InsertThreadShutdown.Add(1)
	go server.ScanExpListRoutine()

	// Run the grpc server on this thread
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	cachegrpc.RegisterCacheServerServer(grpcServer, server.NewServer())
	grpcServer.Serve(lis)

	server.NotifyInsertThreadShutdown <- struct{}{}
	server.InsertThreadShutdown.Wait()
}
