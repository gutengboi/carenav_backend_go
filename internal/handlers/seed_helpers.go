package handlers

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ensureSeeded inserts docs for patientID if no documents already exist for
// them in coll. Used to give every patient the same starter demo data the
// old hardcoded frontend mocks showed, without a separate migration step —
// real writes (e.g. deleting a report, booking a consultation) take over
// from there.
func ensureSeeded(ctx context.Context, coll *mongo.Collection, patientID bson.ObjectID, docs []interface{}) error {
	if len(docs) == 0 {
		return nil
	}
	count, err := coll.CountDocuments(ctx, bson.M{"patientId": patientID})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = coll.InsertMany(ctx, docs)
	return err
}
