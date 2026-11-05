#!/bin/sh
# ==============================================================================
# HADESGATE // ZERO-PII AI PRIVACY GATEWAY & SHIELD
# Autonomous Single-Command Installer (curl -fsSL https://get.hadesgate.dev | sh)
# ==============================================================================

set -e

REPO="hadesgate/hadesgate"
DEFAULT_VERSION="v2.0.0"

# Colors for Industrial Brutalism terminal styling
if [ -t 1 ]; then
  BOLD="\033[1m"
  CYAN="\033[36m"
  GREEN="\033[32m"
  YELLOW="\033[33m"
  RED="\033[31m"
  RESET="\033[0m"
else
  BOLD=""
  CYAN=""
  GREEN=""
  YELLOW=""
  RED=""
  RESET=""
fi

print_banner() {
  printf "${CYAN}${BOLD}"
  printf "==============================================================\n"
  printf "   HADESGATE // ZERO-PII TACTICAL PRIVACY SHIELD             \n"
  printf "   Autonomous Multi-Platform Installer                       \n"
  printf "==============================================================\n"
  printf "${RESET}\n"
}

info() {
  printf "${CYAN}[ INFO ]${RESET} %s\n" "$1"
}

success() {
  printf "${GREEN}[ OK ]${RESET} %s\n" "$1"
}

warn() {
  printf "${YELLOW}[ AVISO ]${RESET} %s\n" "$1"
}

error() {
  printf "${RED}[ ERRO ]${RESET} %s\n" "$1" >&2
  exit 1
}

# 1. Detect Operating System
detect_os() {
  OS_RAW="$(uname -s)"
  case "$OS_RAW" in
    Linux*)  OS="linux" ;;
    Darwin*) OS="darwin" ;;
    *)       error "Sistema operacional não suportado: $OS_RAW. Para Windows, baixe o pacote ZIP em https://github.com/$REPO/releases" ;;
  esac
}

# 2. Detect Hardware Architecture
detect_arch() {
  ARCH_RAW="$(uname -m)"
  case "$ARCH_RAW" in
    x86_64|amd64)   ARCH="amd64" ;;
    arm64|aarch64)  ARCH="arm64" ;;
    *)              error "Arquitetura não suportada: $ARCH_RAW. HadesGate suporta amd64 e arm64." ;;
  esac
}

# 3. Detect Download Tool (curl or wget)
detect_downloader() {
  if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl"
  elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget"
  else
    error "Nem 'curl' nem 'wget' foram encontrados no sistema. Instale um deles para continuar."
  fi
}

download_file() {
  url="$1"
  dest="$2"
  if [ "$DOWNLOADER" = "curl" ]; then
    curl -fsSL "$url" -o "$dest"
  else
    wget -qO "$dest" "$url"
  fi
}

# 4. Resolve Target Release Version
resolve_version() {
  if [ -n "$VERSION" ]; then
    RELEASE_TAG="$VERSION"
    info "Versão solicitada via ambiente: $RELEASE_TAG"
    return
  fi

  info "Verificando última versão estável no GitHub..."
  LATEST_JSON="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null || true)"
  TAG="$(echo "$LATEST_JSON" | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' || true)"

  if [ -n "$TAG" ]; then
    RELEASE_TAG="$TAG"
    info "Versão mais recente identificada: $RELEASE_TAG"
  else
    RELEASE_TAG="$DEFAULT_VERSION"
    warn "Não foi possível consultar API do GitHub (rate limit ou offline). Usando versão padrão: $RELEASE_TAG"
  fi
}

# 5. Main Installation Process
main() {
  print_banner
  detect_os
  detect_arch
  detect_downloader

  info "Plataforma detectada: ${BOLD}$OS/$ARCH${RESET}"
  resolve_version

  TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'hadesgate')"
  trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

  ARCHIVE_NAME="hadesgate-${OS}-${ARCH}.tar.gz"
  DOWNLOAD_URL="https://github.com/$REPO/releases/download/$RELEASE_TAG/$ARCHIVE_NAME"
  CHECKSUM_URL="https://github.com/$REPO/releases/download/$RELEASE_TAG/SHA256SUMS"

  info "Baixando pacote binário: $ARCHIVE_NAME..."
  if ! download_file "$DOWNLOAD_URL" "$TMP_DIR/$ARCHIVE_NAME"; then
    error "Falha ao baixar $DOWNLOAD_URL. Verifique sua conexão ou se a release existe."
  fi
  success "Download concluído."

  info "Verificando integridade criptográfica SHA-256..."
  if download_file "$CHECKSUM_URL" "$TMP_DIR/SHA256SUMS"; then
    EXPECTED_SHA="$(grep "$ARCHIVE_NAME" "$TMP_DIR/SHA256SUMS" | awk '{print $1}' || true)"
    if [ -n "$EXPECTED_SHA" ]; then
      if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_SHA="$(sha256sum "$TMP_DIR/$ARCHIVE_NAME" | awk '{print $1}')"
      elif command -v shasum >/dev/null 2>&1; then
        ACTUAL_SHA="$(shasum -a 256 "$TMP_DIR/$ARCHIVE_NAME" | awk '{print $1}')"
      fi

      if [ -n "$ACTUAL_SHA" ]; then
        if [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
          error "Checksum SHA-256 divergente! Esperado: $EXPECTED_SHA, Obtido: $ACTUAL_SHA"
        fi
        success "Checksum SHA-256 verificado com sucesso."
      fi
    fi
  else
    warn "Arquivo SHA256SUMS não disponível para esta release; prosseguindo sem validação de hash."
  fi

  info "Descompactando binário..."
  tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

  # Choose install directory
  INSTALL_DIR="/usr/local/bin"
  USE_SUDO=""

  if [ -w "$INSTALL_DIR" ]; then
    USE_SUDO=""
  elif command -v sudo >/dev/null 2>&1 && [ "$USER" != "root" ]; then
    USE_SUDO="sudo"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    USE_SUDO=""
  fi

  info "Instalando binário em $INSTALL_DIR/hadesgate..."
  $USE_SUDO mv "$TMP_DIR/hadesgate" "$INSTALL_DIR/hadesgate"
  $USE_SUDO chmod 755 "$INSTALL_DIR/hadesgate"
  success "Binário instalado: $INSTALL_DIR/hadesgate"

  # Prepare local configuration directory
  CONFIG_DIR="$HOME/.hadesgate"
  mkdir -p "$CONFIG_DIR" "$CONFIG_DIR/data"
  if [ ! -f "$CONFIG_DIR/.env" ] && [ -f "$TMP_DIR/.env.example" ]; then
    cp "$TMP_DIR/.env.example" "$CONFIG_DIR/.env"
    info "Criado arquivo de ambiente padrão em: $CONFIG_DIR/.env"
  fi

  # Path warning if installed to ~/.local/bin and not on PATH
  case ":$PATH:" in
    *:"$INSTALL_DIR":*) ;;
    *)
      warn "O diretório $INSTALL_DIR não está no seu PATH!"
      warn "Adicione a seguinte linha ao seu ~/.bashrc ou ~/.zshrc:"
      warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
      ;;
  esac

  printf "\n${GREEN}${BOLD}"
  printf "==============================================================\n"
  printf "   [ INSTALAÇÃO CONCLUÍDA COM SUCESSO! ]                      \n"
  printf "==============================================================\n"
  printf "${RESET}\n"
  printf "Para iniciar o HadesGate, execute:\n"
  printf "  ${CYAN}${BOLD}hadesgate${RESET}\n\n"
  printf "Documentação & Console Web:\n"
  printf "  ${BOLD}http://localhost:8080${RESET}\n\n"
}

main "$@"

