package handlers

import (
	"net/http"

	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// PharmacyHandler serves the nearby pharmacy catalog.
type PharmacyHandler struct {
	Pharmacies *mongo.Collection
}

func NewPharmacyHandler(db *mongo.Database) *PharmacyHandler {
	return &PharmacyHandler{Pharmacies: db.Collection(models.PharmaciesCollection)}
}

// ListPharmacies godoc
// @Summary		List nearby pharmacies
// @Tags			pharmacies
// @Produce		json
// @Success		200	{array}		models.Pharmacy
// @Router			/api/pharmacies [get]
func (h *PharmacyHandler) ListPharmacies(c *gin.Context) {
	ctx, cancel := reqCtx(c)
	defer cancel()

	cursor, err := h.Pharmacies.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load pharmacies."})
		return
	}
	defer cursor.Close(ctx)

	pharmacies := []models.Pharmacy{}
	if err := cursor.All(ctx, &pharmacies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load pharmacies."})
		return
	}

	c.JSON(http.StatusOK, pharmacies)
}
