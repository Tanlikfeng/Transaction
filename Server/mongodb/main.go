package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Account struct {
	ID     int64 `bson:"id"`
	Amount int64 `bson:"amount"`
}

var (
	mongoURI       = "mongodb://localhost:27017"
	databaseName   = "two-phase-commit"
	collectionName = "bank_account"
)

func CheckAmount(client *mongo.Client, ctx context.Context, accountID int64) (int64, int64, error) {

	fmt.Print("inside checkamount\n")
	fmt.Print(accountID)

	collection := client.Database(databaseName).Collection(collectionName)

	filter := bson.D{{"id", accountID}}
	// fmt.Print("\n", filter)

	var account Account
	result := collection.FindOne(context.TODO(), filter).Decode(&account)
	if result != nil {
		if result == mongo.ErrNoDocuments {
			fmt.Println("\nAccount not found.")
			return 0, 0, nil
		}
		fmt.Println("Error:", result)
		return 0, 0, result
	}

	fmt.Println("\nAccount found:", account)
	return account.ID, account.Amount, nil
}

func SendTransaction(client *mongo.Client, ctx context.Context, accountID int64, amount int64) error {

	fmt.Print("Inside sendTransaction\n")
	collection := client.Database(databaseName).Collection(collectionName)

	filter := bson.M{"id": accountID}
	update := bson.M{"$inc": bson.M{"amount": -amount}}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("insufficient funds or account not found")
	}
	return nil
}

func isConnected(client *mongo.Client, ctx context.Context) bool {
	err := client.Ping(ctx, nil)
	if err != nil {
		fmt.Println("Not connected to MongoDB.")
		return false
	}
	fmt.Println("Connected to MongoDB.")
	return true
}

var ClientOptions = options.Client().ApplyURI(mongoURI)
var Client, _ = mongo.NewClient(ClientOptions)
