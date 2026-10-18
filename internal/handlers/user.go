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

// UserHandler mirrors src/controllers/user.controller.ts.
type UserHandler struct {
	Users *mongo.Collection
}

func NewUserHandler(db *mongo.Database) *UserHandler {
	return &UserHandler{Users: db.Collection(models.UsersCollection)}
}

// GetUser godoc
// @Summary		Get a user by ID
// @Tags			users
// @Produce		json
// @Param			id	path		string	true	"User ID"
// @Success		200	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]string
// @Router			/api/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User does not exist"})
		return
	}

	if current := middleware.CurrentUser(c); current == nil || current.ID != oid {
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	var user models.User
	if err := h.Users.FindOne(ctx, bson.M{"_id": oid}).Decode(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User does not exist"})
		return
	}

	c.JSON(http.StatusOK, userResponse(&user))
}

// doctorAccountResponse is what the admin UI's doctor list renders.
type doctorAccountResponse struct {
	ID        bson.ObjectID `json:"id"`
	Username  string        `json:"username"`
	Email     string        `json:"email"`
	CreatedAt time.Time     `json:"createdAt"`
}

// ListDoctorAccounts godoc
// @Summary		List all doctor accounts (admin only)
// @Tags			admin
// @Produce		json
// @Success		200	{array}		doctorAccountResponse
// @Router			/api/admin/doctors [get]
func (h *UserHandler) ListDoctorAccounts(c *gin.Context) {
	ctx, cancel := reqCtx(c)
	defer cancel()

	cursor, err := h.Users.Find(ctx, bson.M{"role": models.RoleDoctor})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load doctors."})
		return
	}
	defer cursor.Close(ctx)

	users := []models.User{}
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load doctors."})
		return
	}

	responses := make([]doctorAccountResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, doctorAccountResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			CreatedAt: u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, responses)
}

// DeleteUser godoc
// @Summary		Delete a user by ID
// @Tags			users
// @Produce		json
// @Param			id	path		string	true	"User ID"
// @Success		200	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router			/api/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	oid, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	if current := middleware.CurrentUser(c); current == nil || current.ID != oid {
		c.JSON(http.StatusForbidden, gin.H{"message": "Access denied."})
		return
	}

	ctx, cancel := reqCtx(c)
	defer cancel()

	res, err := h.Users.DeleteOne(ctx, bson.M{"_id": oid})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "An error occurred", "error": err.Error()})
		return
	}
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully Deleted"})
}
