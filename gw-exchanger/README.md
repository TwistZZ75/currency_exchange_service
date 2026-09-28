# все курсы
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeRates

# курсы пары валют
grpcurl -plaintext -d '{\"from_currency\":\"USD\",\"to_currency\":\"RUB\"}' localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency