package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AllergyRecord holds one patient's selected allergies. One document per
// patient, upserted by the SaveAllergies handler.
type AllergyRecord struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"-"`
	PatientID bson.ObjectID `bson:"patientId" json:"-"`
	Items     []string      `bson:"items" json:"items"`
	UpdatedAt time.Time     `bson:"updatedAt" json:"-"`
}

const AllergiesCollection = "allergies"
