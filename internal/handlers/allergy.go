package handlers

import (
	"net/http"
	"time"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AllergyHandler serves and persists a patient's selected allergies.
type AllergyHandler struct {
	Allergies *mongo.Collection
}

func NewAllergyHandler(db *mongo.Database) *AllergyHandler {
	return &AllergyHandler{Allergies: db.Collection(models.AllergiesCollection)}
}

// GetAllergies godoc
// @Summary		Get the current patient's saved allergies
// @Tags			allergies
// @Produce		json
// @Success		200	{object}	map[string]interface{}
// @Failure		401	{object}	map[string]string
// @Router			/api/allergies [get]
func (h *AllergyHandler) GetAllergies(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var record models.AllergyRecord
	err := h.Allergies.FindOne(ctx, bson.M{"patientId": current.ID}).Decode(&record)
	if err != nil && err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load allergies."})
		return
	}
	if record.Items == nil {
		record.Items = []string{}
	}

	c.JSON(http.StatusOK, gin.H{"items": record.Items})
}

type saveAllergiesRequest struct {
	Items []string `json:"items"`
}

// SaveAllergies godoc
// @Summary		Save the current patient's selected allergies
// @Tags			allergies
// @Accept			json
// @Produce		json
// @Param			body	body		saveAllergiesRequest	true	"Selected allergy items"
// @Success		200		{object}	map[string]interface{}
// @Failure		401		{object}	map[string]string
// @Router			/api/allergies [put]
func (h *AllergyHandler) SaveAllergies(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	var req saveAllergiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "items must be a list of strings."})
		return
	}
	if req.Items == nil {
		req.Items = []string{}
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	_, err := h.Allergies.UpdateOne(
		ctx,
		bson.M{"patientId": current.ID},
		bson.M{"$set": bson.M{"items": req.Items, "updatedAt": time.Now()}},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to save allergies."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": req.Items})
}
