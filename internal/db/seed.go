package db

import (
	"context"
	"time"

	"carenav_backend_go/internal/cryptoutil"
	"carenav_backend_go/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// SeedAdminIfEmpty creates the single shared admin account from env-provided
// credentials the first time the app starts with no admin user yet. If
// email/password aren't configured, or an admin already exists, it's a
// no-op — there's no public signup path for the admin role.
func SeedAdminIfEmpty(ctx context.Context, database *mongo.Database, email, password string) error {
	if email == "" || password == "" {
		return nil
	}

	users := database.Collection(models.UsersCollection)
	count, err := users.CountDocuments(ctx, bson.M{"role": models.RoleAdmin})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hashed, err := cryptoutil.Hash(password)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = users.InsertOne(ctx, models.User{
		Email:      email,
		Password:   hashed,
		Username:   "Admin",
		IsVerified: true,
		Role:       models.RoleAdmin,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	return err
}

// SeedIfEmpty populates the global catalog collections (doctors, medicines,
// pharmacies) with starter demo data the first time the app connects to an
// empty database. It is a no-op on every subsequent run.
func SeedIfEmpty(ctx context.Context, database *mongo.Database) error {
	if err := seedDoctors(ctx, database); err != nil {
		return err
	}
	if err := seedMedicines(ctx, database); err != nil {
		return err
	}
	if err := seedPharmacies(ctx, database); err != nil {
		return err
	}
	return nil
}

func seedIfEmpty(ctx context.Context, coll *mongo.Collection, docs []interface{}) error {
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = coll.InsertMany(ctx, docs)
	return err
}

func seedDoctors(ctx context.Context, database *mongo.Database) error {
	now := time.Now()
	docs := []interface{}{
		models.Doctor{Name: "Dr. Romi Ahmed", Specialty: "Dentist", Rating: 4.8, ImageURL: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRx9_YGXkCZ96VfdvVYVGQSL7Pk3HSaIgmsYw&s", CreatedAt: now},
		models.Doctor{Name: "Dr. Olivia Turner", Specialty: "Dermatologist", Rating: 4.5, ImageURL: "https://img.freepik.com/free-photo/smiling-asian-male-doctor-pointing-upwards_1262-18321.jpg", CreatedAt: now},
		models.Doctor{Name: "Dr. John Doe", Specialty: "General Medical Practice", Rating: 4.7, ImageURL: "https://www.shutterstock.com/image-photo/healthcare-medical-staff-concept-portrait-600nw-2281024823.jpg", CreatedAt: now},
		models.Doctor{Name: "Dr. Sarah Smith", Specialty: "Pediatrician", Rating: 4.9, ImageURL: "https://www.citizenshospitals.com/static/uploads/130789a4-764e-4ee3-88fe-68f9278452d6-1692966652977.png", CreatedAt: now},
	}
	return seedIfEmpty(ctx, database.Collection(models.DoctorsCollection), docs)
}

func seedMedicines(ctx context.Context, database *mongo.Database) error {
	docs := []interface{}{
		models.Medicine{Name: "Paracetamol 500mg", Category: "Pain Relief", Dosage: "1-2 tablets every 4-6 hours", Use: "Relieves mild to moderate pain and reduces fever.", Warning: "Do not exceed 8 tablets in 24 hours.", InStock: true},
		models.Medicine{Name: "Ibuprofen 200mg", Category: "Pain Relief", Dosage: "1 tablet every 6-8 hours with food", Use: "Reduces inflammation, pain, and fever.", Warning: "Avoid on an empty stomach; not for people with ulcers.", InStock: true},
		models.Medicine{Name: "Amoxicillin 500mg", Category: "Antibiotics", Dosage: "1 capsule every 8 hours for 7 days", Use: "Treats bacterial infections such as chest and ear infections.", Warning: "Complete the full course even if you feel better.", InStock: false},
		models.Medicine{Name: "Cetirizine 10mg", Category: "Allergy", Dosage: "1 tablet once daily", Use: "Relieves sneezing, itching, and runny nose from allergies.", Warning: "May cause drowsiness — avoid driving until you know how it affects you.", InStock: true},
		models.Medicine{Name: "Vitamin D3 1000 IU", Category: "Vitamins", Dosage: "1 capsule daily with a meal", Use: "Supports bone health and immune function.", Warning: "Consult a doctor before combining with calcium supplements.", InStock: true},
		models.Medicine{Name: "Omeprazole 20mg", Category: "Digestive", Dosage: "1 capsule daily before breakfast", Use: "Reduces stomach acid for heartburn and acid reflux.", Warning: "Long-term use should be reviewed with your doctor.", InStock: true},
	}
	return seedIfEmpty(ctx, database.Collection(models.MedicinesCollection), docs)
}

func seedPharmacies(ctx context.Context, database *mongo.Database) error {
	docs := []interface{}{
		models.Pharmacy{Name: "HealthFirst Pharmacy", Distance: "0.4 km away", Address: "12 Riverside Road, Lagos", Phone: "+2348012345678", Hours: "Open until 10:00 PM", Open: true},
		models.Pharmacy{Name: "CareWell Drugstore", Distance: "1.1 km away", Address: "45 Allen Avenue, Lagos", Phone: "+2348023456789", Hours: "Open 24 hours", Open: true},
		models.Pharmacy{Name: "City Meds Pharmacy", Distance: "2.3 km away", Address: "8 Marina Street, Lagos", Phone: "+2348034567890", Hours: "Opens at 8:00 AM", Open: false},
	}
	return seedIfEmpty(ctx, database.Collection(models.PharmaciesCollection), docs)
}
