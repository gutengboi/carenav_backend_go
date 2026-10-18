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

// ConsultationHandler serves a patient's appointments (consultations) and
// handles booking/cancelling/rescheduling them, plus a doctor's own
// schedule/patient-queue view.
type ConsultationHandler struct {
	Consultations *mongo.Collection
	Doctors       *mongo.Collection
	Users         *mongo.Collection
}

func NewConsultationHandler(db *mongo.Database) *ConsultationHandler {
	return &ConsultationHandler{
		Consultations: db.Collection(models.ConsultationsCollection),
		Doctors:       db.Collection(models.DoctorsCollection),
		Users:         db.Collection(models.UsersCollection),
	}
}

// consultationResponse is what AppointmentScreen.tsx renders: the
// consultation fields plus the doctor info it needs for the card.
type consultationResponse struct {
	ID           bson.ObjectID             `json:"id"`
	DoctorID     bson.ObjectID             `json:"doctorId"`
	DoctorName   string                    `json:"doctorName"`
	DoctorField  string                    `json:"doctorField"`
	DoctorImage  string                    `json:"doctorImage"`
	DateTime     time.Time                 `json:"dateTime"`
	Status       models.ConsultationStatus `json:"status"`
	CancelReason string                    `json:"cancelReason,omitempty"`
}

// tabStatuses maps the frontend's three tabs onto the backend's four
// consultation statuses — "Upcoming" covers both Pending and Confirmed.
func tabStatuses(tab string) []models.ConsultationStatus {
	switch tab {
	case "Completed":
		return []models.ConsultationStatus{models.ConsultationCompleted}
	case "Canceled":
		return []models.ConsultationStatus{models.ConsultationCancelled}
	default: // "Upcoming" or unset
		return []models.ConsultationStatus{models.ConsultationPending, models.ConsultationConfirmed}
	}
}

// ListConsultations godoc
// @Summary		List the current patient's appointments
// @Tags			consultations
// @Produce		json
// @Param			status	query		string	false	"Upcoming | Completed | Canceled"
// @Success		200		{array}		consultationResponse
// @Failure		401		{object}	map[string]string
// @Router			/api/consultations [get]
func (h *ConsultationHandler) ListConsultations(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	statuses := tabStatuses(c.Query("status"))
	cursor, err := h.Consultations.Find(ctx, bson.M{"patientId": current.ID, "status": bson.M{"$in": statuses}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load appointments."})
		return
	}
	defer cursor.Close(ctx)

	consultations := []models.Consultation{}
	if err := cursor.All(ctx, &consultations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load appointments."})
		return
	}

	responses := make([]consultationResponse, 0, len(consultations))
	for _, consultation := range consultations {
		var doctor models.Doctor
		_ = h.Doctors.FindOne(ctx, bson.M{"_id": consultation.DoctorID}).Decode(&doctor)

		responses = append(responses, consultationResponse{
			ID:           consultation.ID,
			DoctorID:     consultation.DoctorID,
			DoctorName:   doctor.Name,
			DoctorField:  doctor.Specialty,
			DoctorImage:  doctor.ImageURL,
			DateTime:     consultation.DateTime,
			Status:       consultation.Status,
			CancelReason: consultation.CancelReason,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// doctorConsultationResponse is what a doctor's own dashboard renders: the
// consultation fields plus the patient info it needs, instead of doctor info.
type doctorConsultationResponse struct {
	ID           bson.ObjectID             `json:"id"`
	PatientID    bson.ObjectID             `json:"patientId"`
	PatientName  string                    `json:"patientName"`
	DateTime     time.Time                 `json:"dateTime"`
	Status       models.ConsultationStatus `json:"status"`
	CancelReason string                    `json:"cancelReason,omitempty"`
}

// ListMyConsultations godoc
// @Summary		List the current doctor's own schedule/patient queue
// @Tags			consultations
// @Produce		json
// @Param			status	query		string	false	"Upcoming | Completed | Canceled"
// @Success		200		{array}		doctorConsultationResponse
// @Failure		401		{object}	map[string]string
// @Router			/api/doctor/consultations [get]
func (h *ConsultationHandler) ListMyConsultations(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var doctorEntry models.Doctor
	if err := h.Doctors.FindOne(ctx, bson.M{"userId": current.ID}).Decode(&doctorEntry); err != nil {
		// No linked directory entry yet (e.g. account predates this link) —
		// nothing booked against it either, so an empty schedule is correct.
		c.JSON(http.StatusOK, []doctorConsultationResponse{})
		return
	}

	statuses := tabStatuses(c.Query("status"))
	cursor, err := h.Consultations.Find(ctx, bson.M{"doctorId": doctorEntry.ID, "status": bson.M{"$in": statuses}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load schedule."})
		return
	}
	defer cursor.Close(ctx)

	consultations := []models.Consultation{}
	if err := cursor.All(ctx, &consultations); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load schedule."})
		return
	}

	responses := make([]doctorConsultationResponse, 0, len(consultations))
	for _, consultation := range consultations {
		var patient models.User
		_ = h.Users.FindOne(ctx, bson.M{"_id": consultation.PatientID}).Decode(&patient)

		responses = append(responses, doctorConsultationResponse{
			ID:           consultation.ID,
			PatientID:    consultation.PatientID,
			PatientName:  patient.Username,
			DateTime:     consultation.DateTime,
			Status:       consultation.Status,
			CancelReason: consultation.CancelReason,
		})
	}

	c.JSON(http.StatusOK, responses)
}

type createConsultationRequest struct {
	DoctorID string    `json:"doctorId"`
	DateTime time.Time `json:"dateTime"`
}

// CreateConsultation godoc
// @Summary		Book a new appointment
// @Tags			consultations
// @Accept			json
// @Produce		json
// @Param			body	body		createConsultationRequest	true	"Doctor + time slot"
// @Success		201		{object}	consultationResponse
// @Failure		400		{object}	map[string]string
// @Router			/api/consultations [post]
func (h *ConsultationHandler) CreateConsultation(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	var req createConsultationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.DoctorID == "" || req.DateTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"message": "doctorId and dateTime are required."})
		return
	}

	doctorID, err := bson.ObjectIDFromHex(req.DoctorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid doctorId."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var doctor models.Doctor
	if err := h.Doctors.FindOne(ctx, bson.M{"_id": doctorID}).Decode(&doctor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Doctor not found."})
		return
	}

	now := time.Now()
	consultation := models.Consultation{
		PatientID: current.ID,
		DoctorID:  doctorID,
		DateTime:  req.DateTime,
		Status:    models.ConsultationPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	res, err := h.Consultations.InsertOne(ctx, consultation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to book appointment."})
		return
	}
	consultation.ID = res.InsertedID.(bson.ObjectID)

	c.JSON(http.StatusCreated, consultationResponse{
		ID:          consultation.ID,
		DoctorID:    doctorID,
		DoctorName:  doctor.Name,
		DoctorField: doctor.Specialty,
		DoctorImage: doctor.ImageURL,
		DateTime:    consultation.DateTime,
		Status:      consultation.Status,
	})
}

type cancelConsultationRequest struct {
	Reason string `json:"reason"`
}

// CancelConsultation godoc
// @Summary		Cancel an appointment
// @Tags			consultations
// @Accept			json
// @Produce		json
// @Param			id		path		string						true	"Consultation ID"
// @Param			body	body		cancelConsultationRequest	false	"Cancellation reason"
// @Success		200		{object}	map[string]string
// @Failure		404		{object}	map[string]string
// @Router			/api/consultations/{id}/cancel [patch]
func (h *ConsultationHandler) CancelConsultation(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Appointment not found"})
		return
	}

	var req cancelConsultationRequest
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "Appointment was Canceled"
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	res, err := h.Consultations.UpdateOne(
		ctx,
		bson.M{"_id": oid, "patientId": current.ID},
		bson.M{"$set": bson.M{"status": models.ConsultationCancelled, "cancelReason": req.Reason, "updatedAt": time.Now()}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to cancel appointment."})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Appointment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment canceled"})
}

type rescheduleConsultationRequest struct {
	DateTime time.Time `json:"dateTime"`
}

// RescheduleConsultation godoc
// @Summary		Reschedule an appointment
// @Tags			consultations
// @Accept			json
// @Produce		json
// @Param			id		path		string							true	"Consultation ID"
// @Param			body	body		rescheduleConsultationRequest	true	"New date/time"
// @Success		200		{object}	map[string]string
// @Failure		404		{object}	map[string]string
// @Router			/api/consultations/{id}/reschedule [patch]
func (h *ConsultationHandler) RescheduleConsultation(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if current == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authorization required"})
		return
	}

	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Appointment not found"})
		return
	}

	var req rescheduleConsultationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.DateTime.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"message": "dateTime is required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	res, err := h.Consultations.UpdateOne(
		ctx,
		bson.M{"_id": oid, "patientId": current.ID},
		bson.M{"$set": bson.M{"dateTime": req.DateTime, "status": models.ConsultationPending, "updatedAt": time.Now()}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to reschedule appointment."})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Appointment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Appointment rescheduled"})
}
