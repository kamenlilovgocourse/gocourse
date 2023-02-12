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

func main() {
	flag.Parse()
	fmt.Printf("cacheserver running on port %d\n", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server.InsertThreadShutdown.Add(1)
	go server.ScanExpListRoutine()

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	cachegrpc.RegisterCacheServerServer(grpcServer, server.NewServer())
	grpcServer.Serve(lis)

	server.NotifyInsertThreadShutdown <- struct{}{}
	server.InsertThreadShutdown.Wait()
}
