package main

import (
	"context"
	"flag"
	"log/slog"
	"time"

	"github.com/yaninyzwitty/golang-proj-with-db/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", ":50051", "The address to listen on for GRPC requests.")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to connrc: %v", err)
	}

	defer conn.Close()
	client := pb.NewCommerceTransactionsClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.CreateTransaction(ctx, &pb.CreateTransactionRequest{
		Balance: 500,
	})
	if err != nil {
		slog.Error("failed to create a transaction: %v", err)
	}
	slog.Info("Transaction", "your transaction id: ", res.TransactionId)

}
