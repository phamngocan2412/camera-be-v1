package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{userService: s}
}

// GetMe godoc
// @Summary      Get current user profile
// @Description  Get the profile of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.UserProfile
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /api/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// UpdateMe godoc
// @Summary      Update current user profile
// @Description  Update the profile of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      models.UpdateProfileRequest  true  "Update profile request"
// @Success      200      {object}  models.UserProfile
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Router       /api/users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.userService.UpdateProfile(userID, req)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, profile)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Change the password of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      models.ChangePasswordRequest  true  "Change password request"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetInt("user_id")
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.ChangePassword(userID, req); err != nil {
		if err.Error() == "old password incorrect" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "old password incorrect"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to change password"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}
