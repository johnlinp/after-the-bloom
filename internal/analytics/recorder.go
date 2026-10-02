package analytics

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const collectionName = "analytic_events"

type Recorder interface {
	Record(context.Context, Event) error
}

type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, Event) error { return nil }

type MongoRecorder struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoRecorder(ctx context.Context, uri, database string) (*MongoRecorder, error) {
	clientOptions := options.Client().
		ApplyURI(uri).
		SetAppName("after-the-bloom").
		SetMaxPoolSize(10).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	recorder := &MongoRecorder{
		client:     client,
		collection: client.Database(database).Collection(collectionName),
	}
	if err := recorder.createIndexes(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return recorder, nil
}

func (r *MongoRecorder) createIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("created_at_desc"),
		},
		{
			Keys: bson.D{
				{Key: "event_type", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("event_type_created_at_desc"),
		},
	})
	return err
}

func (r *MongoRecorder) Record(ctx context.Context, event Event) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *MongoRecorder) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

func ConnectTimeout() time.Duration { return 10 * time.Second }
