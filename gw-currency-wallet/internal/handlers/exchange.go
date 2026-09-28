package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gw-currency-wallet/internal/middleware"
	"gw-currency-wallet/internal/service"
)

type exchangeReq struct {
	FromCurrency string  `json:"from_currency" binding:"required"`
	ToCurrency   string  `json:"to_currency" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
}

// Rates — GET /api/v1/exchange/rates
func (h *Handlers) Rates(c *gin.Context) {
	rates, err := h.Svc.GetRates(c.Request.Context())
	if err != nil {
		h.Logger.Error("get rates failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve exchange rates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rates": rates})
}

// Exchange — POST /api/v1/exchange
func (h *Handlers) Exchange(c *gin.Context) {
	userID := middleware.UserID(c)

	var req exchangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid currencies"})
		return
	}

	idemp := c.GetHeader("Idempotency-Key")

	w, received, rate, err := h.Svc.Exchange(c.Request.Context(), userID, req.FromCurrency, req.ToCurrency, req.Amount, idemp)
	if err != nil {
		switch err {
		case service.ErrInsufficientFunds, service.ErrInvalidAmount, service.ErrInvalidCurrency, service.ErrInvalidCurrencyPair:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds or invalid currencies"})
		default:
			h.Logger.Error("exchange failed", "error", err, "user_id", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "exchange failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Exchange successful",
		"exchanged_amount": received,
		"rate":             rate,
		"new_balance": gin.H{
			"USD": w.USD,
			"RUB": w.RUB,
			"EUR": w.EUR,
		},
	})
}
