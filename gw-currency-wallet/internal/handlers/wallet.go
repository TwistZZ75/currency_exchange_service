package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gw-currency-wallet/internal/middleware"
	"gw-currency-wallet/internal/service"
)

type amountReq struct {
	Amount   float64 `json:"amount" binding:"required"`
	Currency string  `json:"currency" binding:"required"`
}

// Balance — GET /api/v1/balance
//
// @Summary      Баланс пользователя
// @Tags         wallet
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]map[string]float64 "balance"
// @Failure      401 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/balance [get]
func (h *Handlers) Balance(c *gin.Context) {
	userID := middleware.UserID(c)
	w, err := h.Svc.GetWallet(c.Request.Context(), userID)
	if err != nil {
		h.Logger.Error("get wallet failed", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get balance"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"balance": gin.H{
			"USD": w.USD,
			"RUB": w.RUB,
			"EUR": w.EUR,
		},
	})
}

// Deposit — POST /api/v1/wallet/deposit
//
// @Summary      Пополнение счёта
// @Description  Идемпотентен по заголовку Idempotency-Key. При сумме >= LARGE_TRANSFER_THRESHOLD событие уходит в Kafka.
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key header string false "Опциональный ключ идемпотентности (UUID)"
// @Param        request body amountReq true "Сумма и валюта"
// @Success      200 {object} map[string]interface{} "message + new_balance"
// @Failure      400 {object} map[string]string "error"
// @Failure      401 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/wallet/deposit [post]
func (h *Handlers) Deposit(c *gin.Context) {
	userID := middleware.UserID(c)

	var req amountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount or currency"})
		return
	}

	idemp := c.GetHeader("Idempotency-Key")

	w, err := h.Svc.Deposit(c.Request.Context(), userID, req.Currency, req.Amount, idemp)
	if err != nil {
		switch err {
		case service.ErrInvalidAmount, service.ErrInvalidCurrency:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount or currency"})
		default:
			h.Logger.Error("deposit failed", "error", err, "user_id", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "deposit failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Account topped up successfully",
		"new_balance": gin.H{"USD": w.USD, "RUB": w.RUB, "EUR": w.EUR},
	})
}

// Withdraw — POST /api/v1/wallet/withdraw
//
// @Summary      Вывод средств
// @Description  Идемпотентен по заголовку Idempotency-Key.
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key header string false "Опциональный ключ идемпотентности (UUID)"
// @Param        request body amountReq true "Сумма и валюта"
// @Success      200 {object} map[string]interface{} "message + new_balance"
// @Failure      400 {object} map[string]string "error"
// @Failure      401 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/wallet/withdraw [post]
func (h *Handlers) Withdraw(c *gin.Context) {
	userID := middleware.UserID(c)

	var req amountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid amount"})
		return
	}

	idemp := c.GetHeader("Idempotency-Key")

	w, err := h.Svc.Withdraw(c.Request.Context(), userID, req.Currency, req.Amount, idemp)
	if err != nil {
		switch err {
		case service.ErrInsufficientFunds, service.ErrInvalidAmount, service.ErrInvalidCurrency:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid amount"})
		default:
			h.Logger.Error("withdraw failed", "error", err, "user_id", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "withdraw failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Withdrawal successful",
		"new_balance": gin.H{"USD": w.USD, "RUB": w.RUB, "EUR": w.EUR},
	})
}
