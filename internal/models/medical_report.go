package models

import "go.mongodb.org/mongo-driver/v2/bson"

// MedicalReport is a patient-scoped visit report. Date is kept as a plain
// ISO date string ("2023-03-27") so the existing frontend
// `new Date(dateString)` formatting keeps working unchanged.
type MedicalReport struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PatientID    bson.ObjectID `bson:"patientId" json:"-"`
	Date         string        `bson:"date" json:"date"`
	HospitalName string        `bson:"hospitalName" json:"hospitalName"`
	Reason       string        `bson:"reason" json:"reason"`
	Doctor       string        `bson:"doctor" json:"doctor"`
	Prescription string        `bson:"prescription" json:"prescription"`
}

const MedicalReportsCollection = "medical_reports"
