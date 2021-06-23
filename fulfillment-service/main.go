package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Emoto13/sort_system/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const sortingRobotAddress = "localhost:10000"
const serverPort = "localhost:10001"

func main() {
	sortingRobot, conn := newSortingRobotClient()
	defer conn.Close()

	grpcServer, lis := newFulfillmentServer(sortingRobot)

	fmt.Printf("gRPC server started. Listening on %s\n", serverPort)
	grpcServer.Serve(lis)
}

func newFulfillmentServer(sortingRobot gen.SortingRobotClient) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	gen.RegisterFulfillmentServer(grpcServer, newFulfillmentService(sortingRobot))
	reflection.Register(grpcServer)

	return grpcServer, lis
}

func newSortingRobotClient() (gen.SortingRobotClient, *grpc.ClientConn) {
	conn, err := grpc.Dial(sortingRobotAddress, grpc.WithInsecure())
	for err != nil {
		log.Println("Error while connecting to sorting robot occured: ", err.Error(), "\nTrying again to connect.")
		conn, err = grpc.Dial(sortingRobotAddress, grpc.WithInsecure())
	}

	client := gen.NewSortingRobotClient(conn)
	return client, conn
}
