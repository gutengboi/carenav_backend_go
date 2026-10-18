package handlers

import (
	"net/http"

	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MedicineHandler serves the medicine catalog.
type MedicineHandler struct {
	Medicines *mongo.Collection
}

func NewMedicineHandler(db *mongo.Database) *MedicineHandler {
	return &MedicineHandler{Medicines: db.Collection(models.MedicinesCollection)}
}

// ListMedicines godoc
// @Summary		List the medicine catalog
// @Tags			medicines
// @Produce		json
// @Success		200	{array}		models.Medicine
// @Router			/api/medicines [get]
func (h *MedicineHandler) ListMedicines(c *gin.Context) {
	ctx, cancel := reqCtx(c)
	defer cancel()

	cursor, err := h.Medicines.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medicines."})
		return
	}
	defer cursor.Close(ctx)

	medicines := []models.Medicine{}
	if err := cursor.All(ctx, &medicines); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medicines."})
		return
	}

	c.JSON(http.StatusOK, medicines)
}
