package handlers

import (
	"net/http"
	"time"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// VitalsHandler serves a patient's latest vital signs snapshot.
type VitalsHandler struct {
	Vitals *mongo.Collection
}

func NewVitalsHandler(db *mongo.Database) *VitalsHandler {
	return &VitalsHandler{Vitals: db.Collection(models.VitalsCollection)}
}

// GetVitals godoc
// @Summary		Get the current patient's latest vitals
// @Tags			vitals
// @Produce		json
// @Success		200	{object}	models.Vitals
// @Failure		401	{object}	map[string]string
// @Router			/api/vitals [get]
func (h *VitalsHandler) GetVitals(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var vitals models.Vitals
	err := h.Vitals.FindOne(ctx, bson.M{"patientId": current.ID}).Decode(&vitals)
	if err == mongo.ErrNoDocuments {
		vitals = models.Vitals{
			PatientID:     current.ID,
			Weight:        "75 kg",
			BloodPressure: "120/80 mmHg",
			HeartRate:     "72 bpm",
			BMI:           "24.5",
			UpdatedAt:     time.Now(),
		}
		if _, insErr := h.Vitals.InsertOne(ctx, vitals); insErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load vitals."})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load vitals."})
		return
	}

	c.JSON(http.StatusOK, vitals)
}
