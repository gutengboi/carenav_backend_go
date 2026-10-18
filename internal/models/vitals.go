package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Vitals holds one patient's latest vital signs snapshot. One document per
// patient, lazily seeded on first read.
type Vitals struct {
	ID            bson.ObjectID `bson:"_id,omitempty" json:"-"`
	PatientID     bson.ObjectID `bson:"patientId" json:"-"`
	Weight        string        `bson:"weight" json:"weight"`
	BloodPressure string        `bson:"bloodPressure" json:"bloodPressure"`
	HeartRate     string        `bson:"heartRate" json:"heartRate"`
	BMI           string        `bson:"bmi" json:"bmi"`
	UpdatedAt     time.Time     `bson:"updatedAt" json:"-"`
}

const VitalsCollection = "vitals"
