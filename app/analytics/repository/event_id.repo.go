package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"quickbite/analytics-service/app/analytics/entity"
)

// EventIDsRepo satisfies lib/coreevents.EventDeduper implicitly via Go duck
// typing — no interface implements clause needed, no import of
// lib/coreevents from here.
type EventIDsRepo struct {
	collection *mongo.Collection
}

func NewEventIDsRepo(db *mongo.Database) *EventIDsRepo {
	return &EventIDsRepo{collection: db.Collection(EventIDCollection)}
}

// MarkSeen inserts eventID; the unique index on event_id turns a replay into
// a dup-key error, which we translate to (false, nil) — "not fresh" is not
// an error condition, it's the expected shape of at-least-once delivery.
func (r *EventIDsRepo) MarkSeen(eventID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, entity.EventID{
		EventID:    eventID,
		ReceivedAt: time.Now().UTC(),
	})
	if err == nil {
		return true, nil
	}
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	return false, errors.New("eventids: insert failed: " + err.Error())
}
