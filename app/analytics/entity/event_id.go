package entity

import "time"

// EventID is one document in the event_ids collection — the dedupe ledger.
// A unique index on EventID makes a replayed insert fail with a dup-key
// error instead of a second document; ReceivedAt carries the TTL index (7
// days — see pkg/mongo/indexes.go) so the ledger doesn't grow unbounded.
type EventID struct {
	EventID    string    `bson:"event_id"`
	ReceivedAt time.Time `bson:"received_at"`
}
