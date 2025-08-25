@echo off
echo Starting gRPC server and agent...

:: Параметры сборки
for /f "tokens=1-3 delims=." %%a in ("%date%") do (
    set DD=%%a
    set MM=%%b
    set YYYY=%%c
)
set DATE=%YYYY%-%MM%-%DD%
for /f "delims=" %%i in ('git rev-parse HEAD') do set COMMIT=%%i

:: Собираем сервер
echo Building server...
go build -ldflags "-X main.buildVersion=%VERSION% -X main.buildDate=%DATE% -X main.buildCommit=%COMMIT%" -o .\cmd\server\server.exe .\cmd\server\

:: Собираем агента
echo Building agent...
go build -ldflags "-X main.buildVersion=%VERSION% -X main.buildDate=%DATE% -X main.buildCommit=%COMMIT%" -o .\cmd\agent\agent.exe .\cmd\agent\

:: Настройка окружения (при необходимости)
set DATABASE_DSN=postgres://postgres:pass@localhost:5432?sslmode=disable
:: ВАЖНО: укажите вашу доверенную подсеть, из которой агент стучится к серверу.
:: Пример для домашней сети 192.168.31.0/24 — поменяйте под себя.
set TRUSTED_SUBNET=192.168.31.0/24

:: Запускаем сервер (gRPC) в отдельном окне
start "Server(gRPC)" cmd /k ".\cmd\server\server.exe -mode=grpc -grpc-addr=localhost:9090 -i 10 -f ./metrics/metrics.json -c ./cmd/server/config/config.json -t %TRUSTED_SUBNET%"
set SERVER_PID=%ERRORLEVEL%

:: Ждем 2 секунды, чтобы сервер успел стартовать
timeout /t 2 >nul

:: Запускаем агента (gRPC) в отдельном окне
start "Agent(gRPC)" cmd /k ".\cmd\agent\agent.exe -mode=grpc -grpc-addr=localhost:9090 -c ./cmd/agent/config/config.json"
set AGENT_PID=%ERRORLEVEL%

echo gRPC Server and Agent are running...
echo Server window shows server logs
echo Agent window shows agent logs
echo Press any key to stop all processes...

:: Ждем нажатия любой клавиши
pause > nul

:: Завершаем процессы
echo.
echo Stopping processes...
taskkill /F /FI "WINDOWTITLE eq Server(gRPC)" 2>nul
taskkill /F /FI "WINDOWTITLE eq Agent(gRPC)" 2>nul

echo All processes stopped.
