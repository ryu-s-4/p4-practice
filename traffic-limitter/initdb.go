package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {

	/* connect to the DB */
	uri := "mongodb://127.0.0.1:27017"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	/* delete all objects. */
	collection := client.Database("test").Collection("test")
	res, err := collection.DeleteMany(context.Background(), bson.D{})
	if err != nil {
		log.Fatal("ERROR: failed to delete the entries.")
	}
	log.Println("INFO: deleted", res.DeletedCount, "documents")
	log.Println("INFO: successfully done")
}
