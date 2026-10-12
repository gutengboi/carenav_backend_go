package models

import (
	"time"

	"carenav_backend_go/internal/cryptoutil"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResetToken matches the "resettokens" collection from the original
// Mongoose model (src/models/resetToken.model.ts). The TTL index that
// expires documents after 1 hour already exists on the collection in
// MongoDB itself, so it keeps working without any Go-side code.
type ResetToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Owner     bson.ObjectID `bson:"owner"`
	Token     string        `bson:"token"`
	CreatedAt time.Time     `bson:"createdAt"`
}

const ResetTokensCollection = "resettokens"

// EncryptToken mirrors the Mongoose pre("save") hook that AES-encrypts the
// OTP before it is persisted.
func EncryptToken(otp, secret string) (string, error) {
	return cryptoutil.Encrypt(otp, secret)
}

// CompareToken mirrors resetTokenSchema.methods.compareToken.
func (rt *ResetToken) CompareToken(otp, secret string) bool {
	decrypted, err := cryptoutil.Decrypt(rt.Token, secret)
	if err != nil {
		return false
	}
	return decrypted == otp
}
