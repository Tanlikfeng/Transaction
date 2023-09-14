package main

import (
	pb "Server"
	"context"
	"fmt"
	"net"
	"sync"

	mongodb "Server/mongodb"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedTwoPhaseCommitServerServer
}

type safeMap struct {
	mu  sync.RWMutex
	Map map[uint64]bool
}

func (sm *safeMap) get(key uint64) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.Map[key]
}
func (sm *safeMap) set(key uint64, value bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.Map[key] = value
}

func (sm *safeMap) delete(key uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.Map, key)
}

var paymentMap sync.Map
var accountEntries = safeMap{Map: make(map[uint64]bool)}

// GetLock returns false if the account is locked or not enough money
// otherwise locks the account, stores the payment and returns true
func (s *server) GetLock(ctx context.Context, payment *pb.Payment) (*pb.IsSuccessful, error) {
	accountId := payment.GetFrom()
	locked := accountEntries.get(accountId)
	if locked {
		return &pb.IsSuccessful{Successful: false}, fmt.Errorf("%d is locked", accountId)
	}
	id, amount, err := mongodb.CheckAmount(mongodb.Client, ctx, -payment.GetAmount())
	if err != nil {
		return &pb.IsSuccessful{Successful: false}, nil
	}
	if amount < -payment.GetAmount() {
		return &pb.IsSuccessful{Successful: false}, fmt.Errorf("%d is not enough", accountId)
	}

	accountEntries.set(accountId, true)
	paymentMap.Store(payment.GetId(), mongodb.Account{ID: id, Amount: amount})
	fmt.Println("GetLock")
	return &pb.IsSuccessful{Successful: true}, nil
}

// Abort deletes the payment and unlocks the account
func (s *server) Abort(ctx context.Context, PaymentID *pb.PaymentID) (*pb.IsSuccessful, error) {

	paymentId := PaymentID.GetId()
	p, loaded := paymentMap.LoadAndDelete(paymentId)
	if !loaded {
		return &pb.IsSuccessful{Successful: true}, nil
	}
	accountId := p.(mongodb.Account).ID
	fmt.Println(accountId)
	accountEntries.delete(uint64(accountId))
	fmt.Println("Abort")
	return &pb.IsSuccessful{Successful: true}, nil
}

// Commit sends the payment to the kafka and deletes the payment and unlocks the account
func (s *server) Commit(ctx context.Context, PaymentID *pb.PaymentID) (*pb.IsSuccessful, error) {
	paymentId := PaymentID.GetId()
	storedPayment, ok := paymentMap.LoadAndDelete(paymentId)
	if !ok {
		return &pb.IsSuccessful{Successful: false}, fmt.Errorf("payment %d not found", paymentId)
	}
	accountId := storedPayment.(mongodb.Account).ID
	amount := storedPayment.(mongodb.Account).Amount
	mongodb.SendTransaction(mongodb.Client, ctx, accountId, amount)
	accountEntries.delete(uint64(accountId))
	fmt.Println("Commit")
	return &pb.IsSuccessful{Successful: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Println(err)
	}
	s := grpc.NewServer()
	pb.RegisterTwoPhaseCommitServerServer(s, &server{})
	fmt.Printf("Server is running on port %v\n", lis.Addr())
	if err := s.Serve(lis); err != nil {
		fmt.Println(err)
	}
}
