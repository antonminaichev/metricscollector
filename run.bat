@echo off
echo Starting server and agent...

:: Запускаем сервер в отдельном окне
start "Server" cmd /k "go run cmd/server/main.go"
set SERVER_PID=%ERRORLEVEL%

:: Ждем 2 секунды, чтобы сервер успел запуститься
timeout /t 2

:: Запускаем агента в отдельном окне
start "Agent" cmd /k "go run cmd/agent/main.go"
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