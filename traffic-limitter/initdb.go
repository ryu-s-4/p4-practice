package main

import (
	"context"
	// "encoding/json"
	// "fmt"
	// "io/ioutil"
	"log"
	"time"
	// "net"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/bson"
	// "go.mongodb.org/mongo-driver/bson/bsontype"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	// "go.mongodb.org/mongo-driver/mongo/readpref"
)

// EntryHelper is helper for Entry
type EntryHelper struct {
	ExternEntries         []*ExternEntryHelper         `json:"extern_entries"`
	TableEntries          []*TableEntryHelper          `json:"table_entries"`
	MeterEntries          []*MeterEntryHelper          `json:"meter_entries"`
	CounterEntries        []*CounterEntryHelper        `json:"counter_entries"`
	MulticastGroupEntries []*MulticastGroupEntryHelper `json:"multicast_group_entries"`
	RegisterEntries       []*RegisterEntryHelper       `json:"register_entries"`
	DigestEntries         []*DigestEntryHelper         `json:digest_entries"`
}

// ExternEntryHelper is helper for ExternEntry.
type ExternEntryHelper struct {
	/* TODO */
	dummy int
}

// TableEntryHelper is helper for TableEntry.
type TableEntryHelper struct {
	Table         string                 `json:"table"`
	Match         map[string]interface{} `json:"match"`
	Action_Name   string                 `json:"action_name"`
	Action_Params map[string]interface{} `json:"action_params"`
}

// MeterEntryHelper is helper for MeterEntry.
type MeterEntryHelper struct {
	/* TODO */
	dummy int
}

// CounterEntryHelper is helper for CounterEntry.
type CounterEntryHelper struct {
	Counter string `json:"counter"`
	Index   int64  `json:index"`
}

// MulticastGroupEntryHelper is helper for MulticastGroupEntry
type MulticastGroupEntryHelper struct {
	Multicast_Group_ID uint32           `json:"multicast_group_id"`
	Replicas           []*ReplicaHelper `json:"replicas"`
}

// ReplicaHelper is helper for Replica.
type ReplicaHelper struct {
	Egress_port uint32 `json:"egress_port"`
	Instance    uint32 `json:"instance"`
}

// RegisterEntryHepler is hepler for RegisterEntry.
type RegisterEntryHelper struct {
	/* TODO */
	dummy int
}

// DigestEntryHelper is helper for DigestEntry.
type DigestEntryHelper struct {
	/* TODO */
	dummy int
}

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

	/*
	cur, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		var id primitive.ObjectID
		rvs, err := cur.Current.Values()
		if err != nil {
			log.Fatal(err)
		}
		for _, rv := range rvs {
			switch rv.Type {
			case bsontype.ObjectID:
				id = rv.ObjectID()
			default:
				log.Fatal("ERROR: retrieved document is invalid type.")
			}
			break
		}
		res_delete, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("INFO: %d documents has been deleted.", res_delete.DeletedCount)
	}
	*/
	log.Println("INFO: successfully done")
}
