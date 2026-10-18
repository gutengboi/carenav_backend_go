package handlers

import (
	"net/http"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// PastMedicationHandler serves a patient's medication history.
type PastMedicationHandler struct {
	PastMedications *mongo.Collection
}

func NewPastMedicationHandler(db *mongo.Database) *PastMedicationHandler {
	return &PastMedicationHandler{PastMedications: db.Collection(models.PastMedicationsCollection)}
}

func seedPastMedications(patientID bson.ObjectID) []interface{} {
	return []interface{}{
		models.PastMedication{PatientID: patientID, Name: "Aspirin", Dosage: "75 mg", Duration: "Jan 2023 - Mar 2023", Type: "Tablet", Description: "Used to reduce pain, fever, or inflammation."},
		models.PastMedication{PatientID: patientID, Name: "Ibuprofen", Dosage: "200 mg", Duration: "Jun 2022 - Sep 2022", Type: "Tablet", Description: "Nonsteroidal anti-inflammatory drug used for pain relief."},
		models.PastMedication{PatientID: patientID, Name: "Amoxicillin", Dosage: "500 mg", Duration: "Nov 2021 - Dec 2021", Type: "Capsule", Description: "Antibiotic used to treat bacterial infections."},
		models.PastMedication{PatientID: patientID, Name: "Cetirizine", Dosage: "10 mg", Duration: "Mar 2021 - Apr 2021", Type: "Tablet", Description: "Antihistamine used to relieve allergy symptoms."},
	}
}

// ListPastMedications godoc
// @Summary		List the current patient's past medications
// @Tags			pastMedications
// @Produce		json
// @Success		200	{array}		models.PastMedication
// @Failure		401	{object}	map[string]string
// @Router			/api/past-medications [get]
func (h *PastMedicationHandler) ListPastMedications(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := ensureSeeded(ctx, h.PastMedications, current.ID, seedPastMedications(current.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load past medications."})
		return
	}

	cursor, err := h.PastMedications.Find(ctx, bson.M{"patientId": current.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load past medications."})
		return
	}
	defer cursor.Close(ctx)

	medications := []models.PastMedication{}
	if err := cursor.All(ctx, &medications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load past medications."})
		return
	}

	c.JSON(http.StatusOK, medications)
}
