package models

import "go.mongodb.org/mongo-driver/v2/bson"

// Pharmacy is a global catalog entry (not patient-scoped).
type Pharmacy struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name     string        `bson:"name" json:"name"`
	Distance string        `bson:"distance" json:"distance"`
	Address  string        `bson:"address" json:"address"`
	Phone    string        `bson:"phone" json:"phone"`
	Hours    string        `bson:"hours" json:"hours"`
	Open     bool          `bson:"open" json:"open"`
}

const PharmaciesCollection = "pharmacies"
