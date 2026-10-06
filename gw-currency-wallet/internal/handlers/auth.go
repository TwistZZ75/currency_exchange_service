package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gw-currency-wallet/internal/service"
)

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register — POST /api/v1/register
//
// @Summary      Регистрация нового пользователя
// @Description  Создаёт пользователя и пустой кошелёк. Проверяет уникальность username и email.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerReq true "Данные регистрации"
// @Success      201 {object} map[string]string "message"
// @Failure      400 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/register [post]
func (h *Handlers) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.Svc.Register(c.Request.Context(), req.Username, req.Password, req.Email); err != nil {
		if err == service.ErrUserExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
			return
		}
		h.Logger.Error("register failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registration failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// Login — POST /api/v1/login
//
// @Summary      Авторизация
// @Description  Возвращает JWT-токен. TTL задаётся в конфиге (JWT_TTL).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body loginReq true "Учётные данные"
// @Success      200 {object} map[string]string "token"
// @Failure      400 {object} map[string]string "error"
// @Failure      401 {object} map[string]string "error"
// @Router       /api/v1/login [post]
func (h *Handlers) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	token, err := h.Svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}
		h.Logger.Error("login failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
