package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/yaninyzwitty/golang-proj-with-db/config"
	"github.com/yaninyzwitty/golang-proj-with-db/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type GrpcServer struct {
	pb.UnimplementedCommerceTransactionsServer
	db *pgx.Conn
}

func main() {
	// Load the configuration files
	cfg := config.NewConfig()

	// Parse database configuration
	pgxConfig, err := pgx.ParseConfig(cfg.DATABASE_URL)
	if err != nil {
		slog.Error("error parsing connection configuration", "error", err)
		return
	}

	pgxConfig.RuntimeParams["application_name"] = "docs_simplecrud_gopgx" //for debugging, not really necessary
	conn, err := pgx.ConnectConfig(context.Background(), pgxConfig)
	if err != nil {
		slog.Error("Error connecting to database", "error", err)
		return
	}
	defer conn.Close(context.Background())

	// Set up table
	err = crdbpgx.ExecuteTx(context.Background(), conn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return initTable(context.Background(), tx)
	})
	if err != nil {
		slog.Error("error initializing table", "error", err)
		return
	}

	// Set up gRPC server
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return
	}

	server := grpc.NewServer()
	grpcServer := &GrpcServer{db: conn}
	pb.RegisterCommerceTransactionsServer(server, grpcServer)
	slog.Info("server listening", "address", lis.Addr().String())
	if err := server.Serve(lis); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}

func initTable(ctx context.Context, tx pgx.Tx) error {
	// Drop existing table if it exists
	slog.Info("Dropping existing accounts table if necessary.")
	if _, err := tx.Exec(ctx, "DROP TABLE IF EXISTS accounts"); err != nil {
		return err
	}

	// Create the accounts table
	slog.Info("Creating accounts table.")
	if _, err := tx.Exec(ctx,
		"CREATE TABLE accounts (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), balance INT8)"); err != nil {
		return err
	}
	return nil
}

func (s *GrpcServer) CreateTransaction(ctx context.Context, req *pb.CreateTransactionRequest) (*pb.TransactionResponse, error) {
	newUUID := uuid.New()

	_, err := s.db.Exec(ctx, "INSERT INTO accounts (id, balance) VALUES ($1, $2)", newUUID, req.Balance)
	if err != nil {
		slog.Error("failed to create transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}

	return &pb.TransactionResponse{
		Success:       true,
		Message:       "Transaction created successfully",
		Balance:       req.Balance,
		TransactionId: newUUID.String(),
	}, nil
}

func (s *GrpcServer) UpdateTransaction(ctx context.Context, req *pb.UpdateTransactionRequest) (*pb.TransactionResponse, error) {
	transactionId, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	// Update the transaction in the database

	result, err := s.db.Exec(ctx, "UPDATE accounts SET balance = $1 WHERE id = $2", req.Balance, transactionId)
	if err != nil {
		slog.Error("failed to update transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update transaction: %v", err)
	}
	if result.RowsAffected() == 0 {
		return nil, status.Errorf(codes.NotFound, "Transaction not found")
	}

	return &pb.TransactionResponse{
		Success:       true,
		Message:       "Transaction updated successfully",
		Balance:       req.Balance,
		TransactionId: req.TransactionId,
	}, nil
}

func (s *GrpcServer) DeleteTransaction(ctx context.Context, req *pb.DeleteTransactionRequest) (*pb.DeleteTransactionResponse, error) {
	// parsed the trasaction id
	transactionId, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	result, err := s.db.Exec(ctx, "DELETE FROM accounts WHERE id = $1", transactionId)
	if err != nil {
		slog.Error("failed to delete transaction", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete transaction: %v", err)
	}
	if result.RowsAffected() == 0 {
		return &pb.DeleteTransactionResponse{
			Success: false,
			Message: "Transaction not found",
		}, nil
	}

	return &pb.DeleteTransactionResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	}, nil
}

func (s *GrpcServer) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	// Parse the transaction ID as a UUID
	transactionId, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	// Query the transaction from the database
	row := s.db.QueryRow(ctx, "SELECT balance FROM accounts WHERE id = $1", transactionId)

	var balance int32
	if err := row.Scan(&balance); err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "failed to find transaction")
		}

		slog.Error("failed to get transaction", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to get transaction: %v", err)
	}

	return &pb.GetTransactionResponse{
		Balance:       balance,
		TransactionId: req.TransactionId,
	}, nil
}
