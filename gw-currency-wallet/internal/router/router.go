package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"gw-currency-wallet/internal/auth"
	"gw-currency-wallet/internal/handlers"
	"gw-currency-wallet/internal/middleware"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "gw-currency-wallet/docs"
)

func New(h *handlers.Handlers, jwt *auth.JWTManager, log *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logging(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)

		authed := api.Group("")
		authed.Use(middleware.Auth(jwt))
		{
			authed.GET("/balance", h.Balance)
			authed.POST("/wallet/deposit", h.Deposit)
			authed.POST("/wallet/withdraw", h.Withdraw)
			authed.GET("/exchange/rates", h.Rates)
			authed.POST("/exchange", h.Exchange)
		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}
