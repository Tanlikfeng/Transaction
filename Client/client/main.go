package main

import (
	pb "Client"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type accountData struct {
	serverIP  string
	accountID uint
}

func getLock(amount int64, transactionId uint64, data accountData, wg *sync.WaitGroup, flag *bool, c pb.TwoPhaseCommitServerClient, ctx context.Context) {
	defer wg.Done()

	r, err := c.GetLock(ctx, &pb.Payment{Id: transactionId, From: uint64(data.accountID), Amount: amount})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(data.serverIP, r.GetSuccessful())
	if(*flag == true){
		*flag = r.GetSuccessful()
	}
}

func send_transaction(sender accountData, receiver accountData, amount int64) {
	// setup sender grpc connection
	conn1, err := grpc.Dial(sender.serverIP, grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn1.Close()
	c1 := pb.NewTwoPhaseCommitServerClient(conn1)
	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	//setup receiver grpc connection
	conn2, err := grpc.Dial(receiver.serverIP, grpc.WithInsecure())
	if err != nil {
		fmt.Println(err)
	}
	defer conn2.Close()
	c2 := pb.NewTwoPhaseCommitServerClient(conn1)
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	var senderTransactionId, receiverTransactionId uint64
	rand.Seed(time.Now().UnixNano())
	senderTransactionId = rand.Uint64()
	if sender.serverIP == receiver.serverIP {
		fmt.Println("same server")
		rand.Seed(time.Now().UnixNano())
		receiverTransactionId = rand.Uint64()
	} else {
		receiverTransactionId = senderTransactionId
	}
	fmt.Println(senderTransactionId, receiverTransactionId)
	fmt.Print(123)
	flag := true
	wg := sync.WaitGroup{}
	wg.Add(2)
	go getLock(-amount, senderTransactionId, sender, &wg, &flag, c1, ctx1)
	go getLock(amount, receiverTransactionId, receiver, &wg, &flag, c2, ctx2)
	wg.Wait()
	if flag {
		a, _ := c1.Commit(ctx1, &pb.PaymentID{Id: senderTransactionId})
		fmt.Println(a)
		b, _ := c2.Commit(ctx2, &pb.PaymentID{Id: receiverTransactionId})
		fmt.Println(b)
	} else {
		c1.Abort(ctx1, &pb.PaymentID{Id: senderTransactionId})
		c2.Abort(ctx2, &pb.PaymentID{Id: receiverTransactionId})
	}

}
func main() {
	send_transaction(accountData{"localhost:50051", 1}, accountData{"localhost:50051", 2}, 1)
}
