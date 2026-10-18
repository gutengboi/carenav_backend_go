package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Doctor is a patient-facing browse/book profile. It is intentionally
// separate from the auth User{Role: doctor} accounts (those are for doctor
// login, these are the directory entries shown to patients).
type Doctor struct {
	ID        bson.ObjectID  `bson:"_id,omitempty" json:"_id"`
	UserID    *bson.ObjectID `bson:"userId,omitempty" json:"-"`
	Name      string         `bson:"name" json:"name"`
	Specialty string         `bson:"specialty" json:"specialty"`
	Rating    float64        `bson:"rating" json:"rating"`
	ImageURL  string         `bson:"imageUrl" json:"imageUrl"`
	CreatedAt time.Time      `bson:"createdAt" json:"-"`
}

const DoctorsCollection = "doctors"
