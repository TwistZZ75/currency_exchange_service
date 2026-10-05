# 1. Регистрация
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/register" -ContentType "application/json" -Body '{"username":"user1","password":"pass123","email":"u@example.com"}'

# 2. Логин — получаем токен
$TOKEN = (Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/login" -ContentType "application/json" -Body '{"username":"user1","password":"pass123"}').token
Write-Host "Token length: $($TOKEN.Length)"

# 3. Баланс (изначально нулевой)
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/balance" -Headers @{ Authorization = "Bearer $TOKEN" }

# 4. Пополнение на 1000 USD
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/wallet/deposit" -Headers @{ Authorization = "Bearer $TOKEN" } -ContentType "application/json" -Body '{"amount":1000,"currency":"USD"}'

# 5. Вывод 50 USD
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/wallet/withdraw" -Headers @{ Authorization = "Bearer $TOKEN" } -ContentType "application/json" -Body '{"amount":50,"currency":"USD"}'

# 6. Курсы
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/exchange/rates" -Headers @{ Authorization = "Bearer $TOKEN" }

# 7. Обмен 100 USD → EUR
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/exchange" -Headers @{ Authorization = "Bearer $TOKEN" } -ContentType "application/json" -Body '{"from_currency":"USD","to_currency":"EUR","amount":100}'

# 8. Финальный баланс
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/balance" -Headers @{ Authorization = "Bearer $TOKEN" }