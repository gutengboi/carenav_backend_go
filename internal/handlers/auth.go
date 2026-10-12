package handlers

import (
	"context"
	"net/http"
	"time"
	"unicode"

	"carenav_backend_go/internal/config"
	"carenav_backend_go/internal/cryptoutil"
	"carenav_backend_go/internal/mail"
	"carenav_backend_go/internal/models"
	"carenav_backend_go/internal/streamchat"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AuthHandler bundles the dependencies the Node version pulled in at
// module scope (User/ResetToken models, streamClient, env secrets).
type AuthHandler struct {
	Users       *mongo.Collection
	ResetTokens *mongo.Collection
	Stream      *streamchat.Client
	Mailer      *mail.Mailer
	Cfg         *config.Config
}

func NewAuthHandler(db *mongo.Database, stream *streamchat.Client, mailer *mail.Mailer, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		Users:       db.Collection(models.UsersCollection),
		ResetTokens: db.Collection(models.ResetTokensCollection),
		Stream:      stream,
		Mailer:      mailer,
		Cfg:         cfg,
	}
}

func reqCtx(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), 10*time.Second)
}

func isNumericPin(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// --- register / verify-registration-otp ----------------------------------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// RegisterUser godoc
// @Summary		Register a new user
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		registerRequest	true	"Registration payload"
// @Success		201		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/register [post]
func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" || req.Username == "" || req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "All fields are required, including role."})
		return
	}

	if req.Role != string(models.RoleDoctor) && req.Role != string(models.RolePatient) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid role provided."})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password must be at least 6 characters."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var existing models.User
	err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User already exists."})
		return
	}
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}

	hashedPassword, err := cryptoutil.Encrypt(req.Password, h.Cfg.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}

	now := time.Now()
	newUser := models.User{
		Email:      req.Email,
		Password:   hashedPassword,
		Username:   req.Username,
		IsVerified: false,
		Role:       models.RolePatient,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	res, err := h.Users.InsertOne(ctx, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}
	userID := res.InsertedID.(bson.ObjectID)

	if err := h.Stream.UpsertUser(streamchat.UpsertUserInput{
		ID:    userID.Hex(),
		Name:  req.Username,
		Email: req.Email,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}

	otp := mail.GenerateOTP()
	encryptedOtp, err := models.EncryptToken(otp, h.Cfg.ResetTokenSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}

	if _, err := h.ResetTokens.InsertOne(ctx, models.ResetToken{
		Owner:     userID,
		Token:     encryptedOtp,
		CreatedAt: now,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User registration failed."})
		return
	}

	_ = h.Mailer.Send(
		req.Email,
		"Verify Your Email - OTP",
		"Your OTP for email verification is: "+otp+".",
		"<p>Your OTP for email verification is: <b>"+otp+"</b>.</p>",
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered. OTP sent to your email for verification.",
	})
}

type verifyOtpRequest struct {
	Email string `json:"email"`
	Otp   string `json:"otp"`
}

// VerifyRegistrationOtp godoc
// @Summary		Verify the OTP sent during registration
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		verifyOtpRequest	true	"Email + OTP"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/verify-registration-Otp [post]
func (h *AuthHandler) VerifyRegistrationOtp(c *gin.Context) {
	var req verifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Otp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and OTP are required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var resetToken models.ResetToken
	if err := h.ResetTokens.FindOne(ctx, bson.M{"owner": user.ID}).Decode(&resetToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No OTP found or it has already been used."})
		return
	}

	if !resetToken.CompareToken(req.Otp, h.Cfg.ResetTokenSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid OTP."})
		return
	}

	if _, err := h.Users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"isVerified": true, "updatedAt": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP verification failed."})
		return
	}

	_, _ = h.ResetTokens.DeleteOne(ctx, bson.M{"_id": resetToken.ID})

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified, user registration complete."})
}

// --- login -----------------------------------------------------------------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func userResponse(u *models.User) gin.H {
	return gin.H{
		"_id":        u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"isVerified": u.IsVerified,
		"role":       u.Role,
	}
}

func (h *AuthHandler) signAppToken(u *models.User, includeRole bool) (string, error) {
	claims := jwt.MapClaims{
		"id":    u.ID.Hex(),
		"email": u.Email,
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	if includeRole {
		claims["role"] = u.Role
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.Cfg.JWTSecret))
}

// LoginUser godoc
// @Summary		Log in as a patient
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		loginRequest	true	"Email + password"
// @Success		200		{object}	map[string]interface{}
// @Failure		401		{object}	map[string]string
// @Router			/login [post]
func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Wrong credentials: Invalid email."})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"message": "Account is not verified."})
		return
	}

	decrypted, err := cryptoutil.Decrypt(user.Password, h.Cfg.Secret)
	if err != nil || decrypted != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Wrong password."})
		return
	}

	appToken, err := h.signAppToken(&user, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed."})
		return
	}

	streamToken, err := h.Stream.CreateToken(user.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"appToken":    appToken,
		"streamToken": streamToken,
		"user":        userResponse(&user),
	})
}

// --- forgot / reset password ------------------------------------------------

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword godoc
// @Summary		Request a password reset OTP
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		forgotPasswordRequest	true	"Email"
// @Success		200		{object}	map[string]string
// @Failure		404		{object}	map[string]string
// @Router			/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email is required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	otp := mail.GenerateOTP()

	_, _ = h.ResetTokens.DeleteOne(ctx, bson.M{"owner": user.ID})

	encryptedOtp, err := models.EncryptToken(otp, h.Cfg.ResetTokenSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred during the password reset process."})
		return
	}

	if _, err := h.ResetTokens.InsertOne(ctx, models.ResetToken{
		Owner:     user.ID,
		Token:     encryptedOtp,
		CreatedAt: time.Now(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred during the password reset process."})
		return
	}

	if err := h.Mailer.Send(
		req.Email,
		"Your Password Reset OTP",
		"Your OTP for password reset is: "+otp+". It is valid for 1 hour.",
		"<p>Your OTP for password reset is: <b>"+otp+"</b>.</p><p>It is valid for 1 hour.</p>",
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP email."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to your email."})
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	Otp         string `json:"otp"`
	NewPassword string `json:"newPassword"`
}

// ResetPassword godoc
// @Summary		Reset password using an OTP
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		resetPasswordRequest	true	"Email, OTP, new password"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Otp == "" || req.NewPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email, OTP, and new password are required."})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password must be at least 6 characters."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var resetToken models.ResetToken
	if err := h.ResetTokens.FindOne(ctx, bson.M{"owner": user.ID}).Decode(&resetToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid or expired OTP."})
		return
	}

	if !resetToken.CompareToken(req.Otp, h.Cfg.ResetTokenSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid OTP."})
		return
	}

	hashedPassword, err := cryptoutil.Encrypt(req.NewPassword, h.Cfg.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password reset failed."})
		return
	}

	if _, err := h.Users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"password": hashedPassword, "updatedAt": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password reset failed."})
		return
	}

	_, _ = h.ResetTokens.DeleteOne(ctx, bson.M{"_id": resetToken.ID})

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successful."})
}

// VerifyOtp godoc
// @Summary		Verify a standalone OTP (password/PIN reset flow)
// @Tags			auth
// @Accept			json
// @Produce		json
// @Param			body	body		verifyOtpRequest	true	"Email + OTP"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/verify-otp [post]
func (h *AuthHandler) VerifyOtp(c *gin.Context) {
	var req verifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Otp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and OTP are required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var resetToken models.ResetToken
	if err := h.ResetTokens.FindOne(ctx, bson.M{"owner": user.ID}).Decode(&resetToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No OTP found or it has already been used."})
		return
	}

	if !resetToken.CompareToken(req.Otp, h.Cfg.ResetTokenSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid OTP."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully."})
}

// --- pin ---------------------------------------------------------------

type createPinRequest struct {
	Email string `json:"email"`
	Pin   string `json:"pin"`
}

// CreatePin godoc
// @Summary		Create a 4-digit PIN for a user
// @Tags			pin
// @Accept			json
// @Produce		json
// @Param			body	body		createPinRequest	true	"Email + PIN"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/create-pin [post]
func (h *AuthHandler) CreatePin(c *gin.Context) {
	var req createPinRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Pin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and PIN are required."})
		return
	}

	if !isNumericPin(req.Pin) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "PIN must be a 4-digit numerical value."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	encryptedPin, err := cryptoutil.Encrypt(req.Pin, h.Cfg.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PIN."})
		return
	}

	if _, err := h.Users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"pin": encryptedPin, "updatedAt": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PIN."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PIN created successfully."})
}

// VerifyPin godoc
// @Summary		Verify a user's PIN
// @Tags			pin
// @Accept			json
// @Produce		json
// @Param			body	body		createPinRequest	true	"Email + PIN"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/verify-pin [post]
func (h *AuthHandler) VerifyPin(c *gin.Context) {
	var req createPinRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Pin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and PIN are required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	if user.Pin == "" {
		c.JSON(http.StatusForbidden, gin.H{"message": "No PIN set for this user."})
		return
	}

	decrypted, err := cryptoutil.Decrypt(user.Pin, h.Cfg.Secret)
	if err != nil || decrypted != req.Pin {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid PIN."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PIN verified successfully."})
}

// ForgotPin godoc
// @Summary		Request a PIN reset OTP
// @Tags			pin
// @Accept			json
// @Produce		json
// @Param			body	body		forgotPasswordRequest	true	"Email"
// @Success		200		{object}	map[string]string
// @Failure		404		{object}	map[string]string
// @Router			/forgot-pin [post]
func (h *AuthHandler) ForgotPin(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email is required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	otp := mail.GenerateOTP()

	_, _ = h.ResetTokens.DeleteOne(ctx, bson.M{"owner": user.ID})

	encryptedOtp, err := models.EncryptToken(otp, h.Cfg.ResetTokenSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred during the PIN reset process."})
		return
	}

	if _, err := h.ResetTokens.InsertOne(ctx, models.ResetToken{
		Owner:     user.ID,
		Token:     encryptedOtp,
		CreatedAt: time.Now(),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred during the PIN reset process."})
		return
	}

	if err := h.Mailer.Send(
		req.Email,
		"Your PIN Reset OTP",
		"Your OTP for PIN reset is: "+otp+". It is valid for 1 hour.",
		"<p>Your OTP for PIN reset is: <b>"+otp+"</b>.</p><p>It is valid for 1 hour.</p>",
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send OTP email."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent to your email."})
}

type resetPinRequest struct {
	Email  string `json:"email"`
	Otp    string `json:"otp"`
	NewPin string `json:"newPin"`
}

// ResetPin godoc
// @Summary		Reset a PIN using an OTP
// @Tags			pin
// @Accept			json
// @Produce		json
// @Param			body	body		resetPinRequest	true	"Email, OTP, new PIN"
// @Success		200		{object}	map[string]string
// @Failure		400		{object}	map[string]string
// @Router			/reset-pin [post]
func (h *AuthHandler) ResetPin(c *gin.Context) {
	var req resetPinRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Otp == "" || req.NewPin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email, OTP, and new PIN are required."})
		return
	}

	if !isNumericPin(req.NewPin) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "PIN must be a 4-digit numerical value."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found."})
		return
	}

	var resetToken models.ResetToken
	if err := h.ResetTokens.FindOne(ctx, bson.M{"owner": user.ID}).Decode(&resetToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid or expired OTP."})
		return
	}

	if !resetToken.CompareToken(req.Otp, h.Cfg.ResetTokenSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid OTP."})
		return
	}

	encryptedPin, err := cryptoutil.Encrypt(req.NewPin, h.Cfg.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset PIN."})
		return
	}

	if _, err := h.Users.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"pin": encryptedPin, "updatedAt": time.Now()}}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset PIN."})
		return
	}

	_, _ = h.ResetTokens.DeleteOne(ctx, bson.M{"_id": resetToken.ID})

	c.JSON(http.StatusOK, gin.H{"message": "PIN reset successfully."})
}

// --- doctor --------------------------------------------------------------

type createDoctorRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// CreateDoctor godoc
// @Summary		Create a doctor account
// @Tags			doctor
// @Accept			json
// @Produce		json
// @Param			body	body		createDoctorRequest	true	"Doctor registration payload"
// @Success		201		{object}	map[string]string
// @Failure		409		{object}	map[string]string
// @Router			/create-doctor [post]
func (h *AuthHandler) CreateDoctor(c *gin.Context) {
	var req createDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" || req.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email, username, and password are required."})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Password must be at least 6 characters long."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var existing models.User
	err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Doctor with this email already exists."})
		return
	}
	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An error occurred while creating the doctor account."})
		return
	}

	hashedPassword, err := cryptoutil.Encrypt(req.Password, h.Cfg.Secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An error occurred while creating the doctor account."})
		return
	}

	now := time.Now()
	doctor := models.User{
		Email:      req.Email,
		Username:   req.Username,
		Password:   hashedPassword,
		Role:       models.RoleDoctor,
		IsVerified: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	res, err := h.Users.InsertOne(ctx, doctor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An error occurred while creating the doctor account."})
		return
	}
	doctorID := res.InsertedID.(bson.ObjectID)

	if err := h.Stream.UpsertUser(streamchat.UpsertUserInput{
		ID:    doctorID.Hex(),
		Name:  req.Username,
		Email: req.Email,
		Role:  string(models.RoleDoctor),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An error occurred while creating the doctor account."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Doctor account created successfully."})
}

// LoginDoctor godoc
// @Summary		Log in as a doctor
// @Tags			doctor
// @Accept			json
// @Produce		json
// @Param			body	body		loginRequest	true	"Email + password"
// @Success		200		{object}	map[string]interface{}
// @Failure		401		{object}	map[string]string
// @Router			/doctor-login [post]
func (h *AuthHandler) LoginDoctor(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email and password are required."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password."})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"message": "Account is not verified."})
		return
	}

	if user.Role != models.RoleDoctor {
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied. Not a doctor account."})
		return
	}

	decrypted, err := cryptoutil.Decrypt(user.Password, h.Cfg.Secret)
	if err != nil || decrypted != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid email or password."})
		return
	}

	appToken, err := h.signAppToken(&user, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed."})
		return
	}

	streamToken, err := h.Stream.CreateToken(user.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"appToken":    appToken,
		"streamToken": streamToken,
		"user":        userResponse(&user),
	})
}
