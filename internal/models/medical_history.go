package models

import "go.mongodb.org/mongo-driver/v2/bson"

// MedicalHistoryItem is a patient-scoped past condition/procedure entry.
type MedicalHistoryItem struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	PatientID bson.ObjectID `bson:"patientId" json:"-"`
	Condition string        `bson:"condition" json:"condition"`
	Year      string        `bson:"year" json:"year"`
	Details   string        `bson:"details" json:"details"`
	Icon      string        `bson:"icon" json:"icon"`
}

const MedicalHistoryCollection = "medical_history"
