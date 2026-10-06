package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gw-analytics/internal/storage"
)

type Handlers struct {
	Storage storage.Storage
}

func NewHandlers(st storage.Storage) *Handlers {
	return &Handlers{Storage: st}
}

// parsePeriodRange возвращает (from, to, period), по умолчанию последний час с шагом в 1m
func parsePeriodRange(c *gin.Context) (time.Time, time.Time, string, bool) {
	period := c.DefaultQuery("period", "1m")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var err error

	now := time.Now().UTC()
	if toStr == "" {
		to = now
	} else if to, err = time.Parse(time.RFC3339, toStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' (RFC3339)"})
		return time.Time{}, time.Time{}, "", false
	}

	if fromStr == "" {
		from = to.Add(-time.Hour)
	} else if from, err = time.Parse(time.RFC3339, fromStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' (RFC3339)"})
		return time.Time{}, time.Time{}, "", false
	}

	return from, to, period, true
}

// Counts — GET /api/v1/analytics/events
//
// @Summary      Количество событий по типам/статусам
// @Tags         analytics
// @Produce      json
// @Param        period query string false "Период агрегации: 1m, 5m, 1h, 1d, 1w" default(1m)
// @Param        from   query string false "RFC3339, начало (по умолчанию — час назад)"
// @Param        to     query string false "RFC3339, конец (по умолчанию — сейчас)"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/analytics/events [get]
func (h *Handlers) Counts(c *gin.Context) {
	from, to, period, ok := parsePeriodRange(c)
	if !ok {
		return
	}
	out, err := h.Storage.CountByTypeStatus(c.Request.Context(), from, to, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from":   from,
		"to":     to,
		"period": period,
		"items":  out,
	})
}

// Latency — GET /api/v1/analytics/latency
//
// @Summary      Статистика latency
// @Tags         analytics
// @Produce      json
// @Param        period query string false "Период агрегации" default(1m)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/analytics/latency [get]
func (h *Handlers) Latency(c *gin.Context) {
	from, to, period, ok := parsePeriodRange(c)
	if !ok {
		return
	}
	out, err := h.Storage.LatencyStats(c.Request.Context(), from, to, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from":   from,
		"to":     to,
		"period": period,
		"items":  out,
	})
}

// Errors — GET /api/v1/analytics/errors
//
// @Summary      Частота ошибок
// @Tags         analytics
// @Produce      json
// @Param        period query string false "Период агрегации" default(1m)
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string "error"
// @Failure      500 {object} map[string]string "error"
// @Router       /api/v1/analytics/errors [get]
func (h *Handlers) Errors(c *gin.Context) {
	from, to, period, ok := parsePeriodRange(c)
	if !ok {
		return
	}
	out, err := h.Storage.ErrorStats(c.Request.Context(), from, to, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from":   from,
		"to":     to,
		"period": period,
		"items":  out,
	})
}
