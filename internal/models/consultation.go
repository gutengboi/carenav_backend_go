package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ConsultationStatus mirrors src/models/consultation.model.ts.
// NOTE: handlers/routes for this are intentionally not implemented yet —
// this struct exists so the collection/field shape is ready when we do.
type ConsultationStatus string

const (
	ConsultationPending   ConsultationStatus = "Pending"
	ConsultationConfirmed ConsultationStatus = "Confirmed"
	ConsultationCancelled ConsultationStatus = "Cancelled"
	ConsultationCompleted ConsultationStatus = "Completed"
)

type Consultation struct {
	ID           bson.ObjectID      `bson:"_id,omitempty"`
	PatientID    bson.ObjectID      `bson:"patientId"`
	DoctorID     bson.ObjectID      `bson:"doctorId"`
	DateTime     time.Time          `bson:"dateTime"`
	Status       ConsultationStatus `bson:"status"`
	Notes        string             `bson:"notes,omitempty"`
	CancelReason string             `bson:"cancelReason,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
}

const ConsultationsCollection = "consultations"
