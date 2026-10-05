package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Storage struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewConn(ctx context.Context, uri, dbName, collName string, timeout time.Duration) (*Storage, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	coll := client.Database(dbName).Collection(collName)

	idxCtx, cancelIdx := context.WithTimeout(ctx, timeout)
	defer cancelIdx()
	_, err = coll.Indexes().CreateOne(idxCtx, mongo.IndexModel{
		Keys:    map[string]int{"transaction_id": 1},
		Options: options.Index().SetUnique(true).SetName("uniq_transaction_id"),
	})
	if err != nil {
		return nil, fmt.Errorf("create unique index: %w", err)
	}

	return &Storage{client: client, collection: coll}, nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, nil)
}

func (s *Storage) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
