package models

import "go.mongodb.org/mongo-driver/v2/bson"

// Medicine is a global catalog entry (not patient-scoped).
type Medicine struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name     string        `bson:"name" json:"name"`
	Category string        `bson:"category" json:"category"`
	Dosage   string        `bson:"dosage" json:"dosage"`
	Use      string        `bson:"use" json:"use"`
	Warning  string        `bson:"warning" json:"warning"`
	InStock  bool          `bson:"inStock" json:"inStock"`
}

const MedicinesCollection = "medicines"
