package handlers

import (
	"net/http"

	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MedicalReportHandler serves a patient's visit reports.
type MedicalReportHandler struct {
	MedicalReports *mongo.Collection
}

func NewMedicalReportHandler(db *mongo.Database) *MedicalReportHandler {
	return &MedicalReportHandler{MedicalReports: db.Collection(models.MedicalReportsCollection)}
}

func seedMedicalReports(patientID bson.ObjectID) []interface{} {
	return []interface{}{
		models.MedicalReport{PatientID: patientID, Date: "2023-03-27", HospitalName: "Healthy Family Centre", Reason: "Allergy Checkup", Doctor: "Dr. Jane Doe", Prescription: "Antihistamines"},
		models.MedicalReport{PatientID: patientID, Date: "2023-04-10", HospitalName: "Healthy Family Centre", Reason: "Diabetes Management", Doctor: "Dr. John Smith", Prescription: "Metformin"},
	}
}

// ListMedicalReports godoc
// @Summary		List the current patient's medical reports
// @Tags			medicalReports
// @Produce		json
// @Success		200	{array}		models.MedicalReport
// @Failure		401	{object}	map[string]string
// @Router			/api/medical-reports [get]
func (h *MedicalReportHandler) ListMedicalReports(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	if err := ensureSeeded(ctx, h.MedicalReports, current.ID, seedMedicalReports(current.ID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical reports."})
		return
	}

	cursor, err := h.MedicalReports.Find(ctx, bson.M{"patientId": current.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical reports."})
		return
	}
	defer cursor.Close(ctx)

	reports := []models.MedicalReport{}
	if err := cursor.All(ctx, &reports); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load medical reports."})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// DeleteMedicalReport godoc
// @Summary		Delete one of the current patient's medical reports
// @Tags			medicalReports
// @Produce		json
// @Param			id	path		string	true	"Report ID"
// @Success		200	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router			/api/medical-reports/{id} [delete]
func (h *MedicalReportHandler) DeleteMedicalReport(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Report not found"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	res, err := h.MedicalReports.DeleteOne(ctx, bson.M{"_id": oid, "patientId": current.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete report."})
		return
	}
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Report not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report deleted"})
}
