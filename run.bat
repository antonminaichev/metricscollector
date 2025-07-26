@echo off
echo Starting server and agent...

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

:: Запускаем сервер в отдельном окне
set DATABASE_DSN=postgres://postgres:pass@localhost:5432?sslmode=disable
start "Server" cmd /k ".\cmd\server\server.exe -a=localhost:8080 -i 10 -f ./metrics/metrics.json"
set SERVER_PID=%ERRORLEVEL%

:: Ждем 2 секунды, чтобы сервер успел запуститься
timeout /t 2

:: Запускаем агента в отдельном окне
start "Agent" cmd /k ".\cmd\agent\agent.exe"
set AGENT_PID=%ERRORLEVEL%

echo Server and Agent are running...
echo Server window shows server logs
echo Agent window shows agent logs
echo Press any key to stop all processes...

:: Ждем нажатия любой клавиши
pause > nul

:: Завершаем процессы
echo.
echo Stopping processes...
taskkill /F /FI "WINDOWTITLE eq Server" 2>nul
taskkill /F /FI "WINDOWTITLE eq Agent" 2>nul

echo All processes stopped.