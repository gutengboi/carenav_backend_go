package router

import (
	"net/http"
	"time"

	"carenav_backend_go/docs"
	"carenav_backend_go/internal/handlers"
	"carenav_backend_go/internal/middleware"
	"carenav_backend_go/internal/models"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

// New wires up the same routes as src/index.ts, plus the patient-facing
// health feature resources (doctors, consultations, medicines, lab tests,
// past medications, medical history/reports, allergies, pharmacies,
// vitals) that src/index.ts never had.
func New(
	auth *handlers.AuthHandler,
	user *handlers.UserHandler,
	doctor *handlers.DoctorHandler,
	consultation *handlers.ConsultationHandler,
	medicine *handlers.MedicineHandler,
	labTest *handlers.LabTestHandler,
	pastMedication *handlers.PastMedicationHandler,
	medicalHistory *handlers.MedicalHistoryHandler,
	medicalReport *handlers.MedicalReportHandler,
	allergy *handlers.AllergyHandler,
	pharmacy *handlers.PharmacyHandler,
	vitals *handlers.VitalsHandler,
) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.SecureHeaders())
	r.Use(middleware.RateLimiter(100, 15*time.Minute))

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to my server")
	})

	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	r.POST("/register", auth.RegisterUser)
	r.POST("/verify-registration-Otp", auth.VerifyRegistrationOtp)
	r.POST("/login", auth.LoginUser)
	r.POST("/forgot-password", auth.ForgotPassword)
	r.POST("/verify-otp", auth.VerifyOtp)
	r.POST("/reset-password", auth.ResetPassword)
	r.POST("/create-pin", auth.CreatePin)
	r.POST("/verify-pin", auth.VerifyPin)
	r.POST("/forgot-pin", auth.ForgotPin)
	r.POST("/reset-pin", auth.ResetPin)
	r.POST("/doctor-login", auth.LoginDoctor)

	// NOTE: deliberately diverges from src/routers/user.routes.ts, which had
	// no auth middleware here — these routes now require a bearer token and
	// are restricted to the token's own user (see user.GetUser/DeleteUser).
	usersGroup := r.Group("/api/users")
	usersGroup.Use(middleware.AuthenticateToken(user.Users, auth.Cfg.JWTSecret))
	usersGroup.GET("/:id", user.GetUser)
	usersGroup.DELETE("/:id", user.DeleteUser)

	// apiGroup carries every patient-facing health feature resource behind
	// the same bearer-token middleware proven above for /api/users.
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.AuthenticateToken(user.Users, auth.Cfg.JWTSecret))

	apiGroup.GET("/doctors", doctor.ListDoctors)

	apiGroup.GET("/consultations", consultation.ListConsultations)
	apiGroup.POST("/consultations", consultation.CreateConsultation)
	apiGroup.PATCH("/consultations/:id/cancel", consultation.CancelConsultation)
	apiGroup.PATCH("/consultations/:id/reschedule", consultation.RescheduleConsultation)

	apiGroup.GET("/medicines", medicine.ListMedicines)
	apiGroup.GET("/lab-tests", labTest.ListLabTests)
	apiGroup.GET("/past-medications", pastMedication.ListPastMedications)
	apiGroup.GET("/medical-history", medicalHistory.ListMedicalHistory)

	apiGroup.GET("/medical-reports", medicalReport.ListMedicalReports)
	apiGroup.DELETE("/medical-reports/:id", medicalReport.DeleteMedicalReport)

	apiGroup.GET("/allergies", allergy.GetAllergies)
	apiGroup.PUT("/allergies", allergy.SaveAllergies)

	apiGroup.GET("/pharmacies", pharmacy.ListPharmacies)
	apiGroup.GET("/vitals", vitals.GetVitals)

	// doctorGroup carries the doctor-only self-service endpoints (their own
	// schedule/patient queue), restricted to Role: doctor accounts.
	doctorGroup := r.Group("/api/doctor")
	doctorGroup.Use(
		middleware.AuthenticateToken(user.Users, auth.Cfg.JWTSecret),
		middleware.AuthorizeRole(models.RoleDoctor),
	)
	doctorGroup.GET("/consultations", consultation.ListMyConsultations)

	// adminGroup carries doctor-onboarding, restricted to Role: admin
	// accounts. Previously /create-doctor was a public, unauthenticated
	// route — that was a real gap, now closed.
	adminGroup := r.Group("/api/admin")
	adminGroup.Use(
		middleware.AuthenticateToken(user.Users, auth.Cfg.JWTSecret),
		middleware.AuthorizeRole(models.RoleAdmin),
	)
	adminGroup.POST("/doctors", auth.CreateDoctor)
	adminGroup.GET("/doctors", user.ListDoctorAccounts)

	return r
}
