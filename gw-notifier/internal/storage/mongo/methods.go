package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"gw-notifier/internal/domain"
)

func (s *Storage) SaveEvent(ctx context.Context, ev domain.TransferEvent) (bool, error) {
	filter := bson.M{"transaction_id": ev.TransactionID}
	update := bson.M{"$setOnInsert": ev}

	res, err := s.collection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, fmt.Errorf("mongo upsert: %w", err)
	}
	return res.UpsertedID != nil, nil
}

func (s *Storage) SaveBatch(ctx context.Context, events []domain.TransferEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	models := make([]mongo.WriteModel, 0, len(events))
	for _, ev := range events {
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"transaction_id": ev.TransactionID}).
			SetUpdate(bson.M{"$setOnInsert": ev}).
			SetUpsert(true))
	}

	res, err := s.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	if err != nil {
		if res != nil {
			return int(res.UpsertedCount), fmt.Errorf("bulk write partial: %w", err)
		}
		return 0, fmt.Errorf("bulk write: %w", err)
	}
	return int(res.UpsertedCount), nil
}
