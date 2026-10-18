package handlers

import (
	"net/http"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// LabTestHandler serves a patient's lab test results.
type LabTestHandler struct {
	LabTests *mongo.Collection
}

func NewLabTestHandler(db *mongo.Database) *LabTestHandler {
	return &LabTestHandler{LabTests: db.Collection(models.LabTestsCollection)}
}

func seedLabTests(patientID bson.ObjectID) []interface{} {
	return []interface{}{
		models.LabTest{PatientID: patientID, Name: "Complete Blood Count (CBC)", Date: "15 Aug 2024", Result: "All values within range", Status: "Normal"},
		models.LabTest{PatientID: patientID, Name: "Liver Function Test", Date: "12 Jul 2024", Result: "Elevated ALT levels", Status: "Elevated"},
		models.LabTest{PatientID: patientID, Name: "Fasting Blood Sugar", Date: "10 Jun 2024", Result: "95 mg/dL", Status: "Normal"},
		models.LabTest{PatientID: patientID, Name: "Vitamin D", Date: "2 May 2024", Result: "18 ng/mL — below range", Status: "Low"},
		models.LabTest{PatientID: patientID, Name: "Lipid Panel", Date: "20 Mar 2024", Result: "LDL slightly elevated", Status: "Elevated"},
	}
}

// ListLabTests godoc
// @Summary		List the current patient's lab test results
// @Tags			labTests
// @Produce		json
// @Success		200	{array}		models.LabTest
// @Failure		401	{object}	map[string]string
// @Router			/api/lab-tests [get]
func (h *LabTestHandler) ListLabTests(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := ensureSeeded(ctx, h.LabTests, current.ID, seedLabTests(current.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load lab tests."})
		return
	}

	cursor, err := h.LabTests.Find(ctx, bson.M{"patientId": current.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load lab tests."})
		return
	}
	defer cursor.Close(ctx)

	labTests := []models.LabTest{}
	if err := cursor.All(ctx, &labTests); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load lab tests."})
		return
	}

	c.JSON(http.StatusOK, labTests)
}
