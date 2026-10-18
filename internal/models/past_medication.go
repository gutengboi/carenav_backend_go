package models

import "go.mongodb.org/mongo-driver/v2/bson"

// PastMedication is a patient-scoped medication history entry.
type PastMedication struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	PatientID   bson.ObjectID `bson:"patientId" json:"-"`
	Name        string        `bson:"name" json:"name"`
	Dosage      string        `bson:"dosage" json:"dosage"`
	Duration    string        `bson:"duration" json:"duration"`
	Type        string        `bson:"type" json:"type"`
	Description string        `bson:"description" json:"description"`
}

const PastMedicationsCollection = "past_medications"
