package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nyaruka/phonenumbers"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RegisterRequest  true  "Registration request"
// @Success      201      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Phone Number
	var parsedNumber *phonenumbers.PhoneNumber
	var err error

	// Check if CountryCode looks like a calling code (starts with +)
	if len(req.CountryCode) > 0 && req.CountryCode[0] == '+' {
		// Concatenate: +84 + 090... -> +84090...
		// parse with default region empty, trusting the + prefix
		fullNumber := req.CountryCode + req.PhoneNumber
		parsedNumber, err = phonenumbers.Parse(fullNumber, "")
	} else {
		// Treat CountryCode as Region Code (e.g. "VN")
		parsedNumber, err = phonenumbers.Parse(req.PhoneNumber, req.CountryCode)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number format"})
		return
	}
	if !phonenumbers.IsValidNumber(parsedNumber) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number"})
		return
	}

	// Format phone number to E.164 standard before saving
	formattedNum := phonenumbers.Format(parsedNumber, phonenumbers.E164)
	req.PhoneNumber = formattedNum

	res, err := h.service.Register(req.Email, req.Password, req.FirstName, req.LastName, req.PhoneNumber, req.CountryCode)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		} else if err.Error() == "phone number already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "phone number already exists"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, res)
}

// Login godoc
// @Summary      Login user
// @Description  Login with email and password to get JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "Login request"
// @Success      200      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, res)
}
