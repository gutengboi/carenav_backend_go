package models

import "go.mongodb.org/mongo-driver/v2/bson"

// LabTest is a patient-scoped lab result entry.
type LabTest struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	PatientID bson.ObjectID `bson:"patientId" json:"-"`
	Name      string        `bson:"name" json:"name"`
	Date      string        `bson:"date" json:"date"`
	Result    string        `bson:"result" json:"result"`
	Status    string        `bson:"status" json:"status"`
}

const LabTestsCollection = "lab_tests"
