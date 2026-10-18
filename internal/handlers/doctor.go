package handlers

import (
	"net/http"

	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// DoctorHandler serves the patient-facing doctor directory.
type DoctorHandler struct {
	Doctors *mongo.Collection
}

func NewDoctorHandler(db *mongo.Database) *DoctorHandler {
	return &DoctorHandler{Doctors: db.Collection(models.DoctorsCollection)}
}

// ListDoctors godoc
// @Summary		List doctors available for consultation
// @Tags			doctors
// @Produce		json
// @Success		200	{array}		models.Doctor
// @Router			/api/doctors [get]
func (h *DoctorHandler) ListDoctors(c *gin.Context) {
	ctx, cancel := reqCtx(c)
	defer cancel()

	cursor, err := h.Doctors.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load doctors."})
		return
	}
	defer cursor.Close(ctx)

	doctors := []models.Doctor{}
	if err := cursor.All(ctx, &doctors); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load doctors."})
		return
	}

	c.JSON(http.StatusOK, doctors)
}
