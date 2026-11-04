// Package mongo wraps the official driver's connection lifecycle. It knows
// nothing about collections, indexes, or any app-specific schema — that
// knowledge belongs to app/<module>/repository.
package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

func Connect(ctx context.Context, uri, dbName string) (*Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{client: client, db: client.Database(dbName)}, nil
}

func (c *Client) Database() *mongo.Database {
	return c.db
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
