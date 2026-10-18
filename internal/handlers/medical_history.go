package handlers

import (
	"net/http"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MedicalHistoryHandler serves a patient's past conditions and procedures.
type MedicalHistoryHandler struct {
	MedicalHistory *mongo.Collection
}

func NewMedicalHistoryHandler(db *mongo.Database) *MedicalHistoryHandler {
	return &MedicalHistoryHandler{MedicalHistory: db.Collection(models.MedicalHistoryCollection)}
}

func seedMedicalHistory(patientID bson.ObjectID) []interface{} {
	return []interface{}{
		models.MedicalHistoryItem{PatientID: patientID, Condition: "Hypertension", Year: "2021", Details: "Diagnosed in 2021. Currently on medication to control blood pressure.", Icon: "pulse-outline"},
		models.MedicalHistoryItem{PatientID: patientID, Condition: "Appendectomy", Year: "2019", Details: "Surgery in 2019 to remove the appendix. No complications.", Icon: "medkit-outline"},
		models.MedicalHistoryItem{PatientID: patientID, Condition: "Asthma", Year: "Childhood", Details: "Diagnosed in childhood. Uses an inhaler for symptom control.", Icon: "fitness-outline"},
		models.MedicalHistoryItem{PatientID: patientID, Condition: "Allergy to Penicillin", Year: "2015", Details: "Severe allergic reaction to penicillin antibiotics. Avoid all penicillin-class drugs.", Icon: "alert-circle-outline"},
	}
}

// ListMedicalHistory godoc
// @Summary		List the current patient's medical history
// @Tags			medicalHistory
// @Produce		json
// @Success		200	{array}		models.MedicalHistoryItem
// @Failure		401	{object}	map[string]string
// @Router			/api/medical-history [get]
func (h *MedicalHistoryHandler) ListMedicalHistory(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := ensureSeeded(ctx, h.MedicalHistory, current.ID, seedMedicalHistory(current.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical history."})
		return
	}

	cursor, err := h.MedicalHistory.Find(ctx, bson.M{"patientId": current.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical history."})
		return
	}
	defer cursor.Close(ctx)

	history := []models.MedicalHistoryItem{}
	if err := cursor.All(ctx, &history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical history."})
		return
	}

	c.JSON(http.StatusOK, history)
}
