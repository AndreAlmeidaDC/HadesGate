<#
.SYNOPSIS
    Ativa o redirecionamento local para o HadesGate na sessao atual do PowerShell.

.DESCRIPTION
    Define as variaveis padrao de Base URL reconhecidas por SDKs e ferramentas
    (OpenAI, Anthropic, Google Gemini e HTTP proxy) apontando para o HadesGate em localhost:8080.

.EXAMPLE
    . .\scripts\enable-hadesgate.ps1
#>

[CmdletBinding()]
param(
    [string]$GatewayUrl = "http://localhost:8080",
    [string]$ProxyKey = "hadesgate_proxy_key_2026"
)

# 1. Base URLs diretas dos provedores LLM
$env:OPENAI_BASE_URL    = "$GatewayUrl/v1"
$env:ANTHROPIC_BASE_URL = "$GatewayUrl"
$env:GEMINI_BASE_URL    = "$GatewayUrl"

# 2. Chaves de autenticacao de proxy (fallback caso seu client nao envie uma)
if (-not $env:HADESGATE_PROXY_KEY) {
    $env:HADESGATE_PROXY_KEY = $ProxyKey
}

# 3. Headers padrao de ambiente para CLI e SDKs
$env:OPENAI_API_BASE = "$GatewayUrl/v1"
$env:ANTHROPIC_API_URL = "$GatewayUrl"

Write-Host "==========================================================" -ForegroundColor Magenta
Write-Host " [HADESGATE] REDIRECIONAMENTO LOCAL ATIVADO COM SUCESSO    " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Magenta
Write-Host " Target Gateway:      $GatewayUrl" -ForegroundColor White
Write-Host " OpenAI Base URL:     $env:OPENAI_BASE_URL" -ForegroundColor Gray
Write-Host " Anthropic Base URL:  $env:ANTHROPIC_BASE_URL" -ForegroundColor Gray
Write-Host " Gemini Base URL:     $env:GEMINI_BASE_URL" -ForegroundColor Gray
Write-Host " Dashboard:           $GatewayUrl" -ForegroundColor Gray
Write-Host "----------------------------------------------------------" -ForegroundColor DarkGray
Write-Host " Prompts enviados por SDKs/ferramentas nesta sessao serao" -ForegroundColor Yellow
Write-Host " sanitizados (Zero-PII) antes de atingir a nuvem externa." -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Magenta

