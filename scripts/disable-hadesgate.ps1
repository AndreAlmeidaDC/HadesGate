<#
.SYNOPSIS
    Desativa o redirecionamento do HadesGate na sessao atual do PowerShell.

.DESCRIPTION
    Remove as variaveis de Base URL e restaura a comunicacao direta com as APIs
    oficiais na nuvem (OpenAI, Anthropic e Google Gemini).

.EXAMPLE
    . .\scripts\disable-hadesgate.ps1
#>

# 1. Remove variaveis de Base URL
Remove-Item Env:\OPENAI_BASE_URL -ErrorAction SilentlyContinue
Remove-Item Env:\OPENAI_API_BASE -ErrorAction SilentlyContinue
Remove-Item Env:\ANTHROPIC_BASE_URL -ErrorAction SilentlyContinue
Remove-Item Env:\ANTHROPIC_API_URL -ErrorAction SilentlyContinue
Remove-Item Env:\GEMINI_BASE_URL -ErrorAction SilentlyContinue

Write-Host "==========================================================" -ForegroundColor DarkGray
Write-Host " [HADESGATE] REDIRECIONAMENTO LOCAL DESATIVADO            " -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor DarkGray
Write-Host " Sessao restaurada para as rotas e upstreams oficiais:" -ForegroundColor White
Write-Host " - OpenAI:    https://api.openai.com/v1" -ForegroundColor Gray
Write-Host " - Anthropic: https://api.anthropic.com" -ForegroundColor Gray
Write-Host " - Gemini:    https://generativelanguage.googleapis.com" -ForegroundColor Gray
Write-Host "==========================================================" -ForegroundColor DarkGray

