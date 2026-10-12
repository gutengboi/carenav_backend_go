package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type UserRole string

const (
	RoleDoctor  UserRole = "doctor"
	RolePatient UserRole = "patient"
)

// User matches the documents in the "users" collection created by the
// original Mongoose model (src/models/user.model.ts). Field names mirror
// the Mongoose schema exactly so existing data keeps working.
type User struct {
	ID                   bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	Username             string        `bson:"username" json:"username"`
	Email                string        `bson:"email" json:"email"`
	Password             string        `bson:"password" json:"-"`
	IsVerified           bool          `bson:"isVerified" json:"isVerified"`
	Role                 UserRole      `bson:"role" json:"role"`
	PasswordResetToken   string        `bson:"passwordResetToken,omitempty" json:"-"`
	PasswordResetExpires *time.Time    `bson:"passwordResetExpires,omitempty" json:"-"`
	Pin                  string        `bson:"pin,omitempty" json:"-"`
	CreatedAt            time.Time     `bson:"createdAt" json:"-"`
	UpdatedAt            time.Time     `bson:"updatedAt" json:"-"`
}

const UsersCollection = "users"
