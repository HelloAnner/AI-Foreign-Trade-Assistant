@echo off
setlocal
cd /d "%~dp0"
if not exist "AI_Trade_Assistant_Windows.exe" (
  echo [ERROR] Missing AI_Trade_Assistant_Windows.exe in %cd%.
  pause
  exit /b 1
)
AI_Trade_Assistant_Windows.exe
