package main

import (
	"fmt"
	_ "gw-currency-wallet/docs"
	"os"
)

// @title           gw-currency-wallet API
// @version         1.0
// @description     REST API кошелька с обменом валют (USD/RUB/EUR).
// @description     Поддерживает регистрацию, логин (JWT), баланс, пополнение, вывод, обмен валют.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите "Bearer <token>", полученный при логине через POST /api/v1/login
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gw-currency-wallet: %v\n", err)
		os.Exit(1)
	}
}
