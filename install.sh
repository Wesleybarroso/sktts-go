#!/usr/bin/env bash

set -e

APP_DIR="/opt/xtts"
API_DIR="$APP_DIR/api"
BIN="$APP_DIR/sktts-go"
SERVICE="/etc/systemd/system/sktts-go.service"

echo "=========================================="
echo "        SKTTS-Go INSTALLER"
echo "=========================================="

if [ "$(id -u)" -ne 0 ]; then
    echo "ERRO: execute como root."
    exit 1
fi

echo "[1/9] Verificando sistema..."

if [ -f /etc/os-release ]; then
    . /etc/os-release
    echo "Sistema: $PRETTY_NAME"
else
    echo "ERRO: não foi possível identificar o sistema."
    exit 1
fi

echo "[2/9] Instalando dependências..."

apt-get update
apt-get install -y \
    ffmpeg \
    curl \
    ca-certificates \
    git \
    build-essential

echo "[3/9] Verificando Go..."

if command -v go >/dev/null 2>&1; then
    echo "Go encontrado:"
    go version
else
    echo "ERRO: Go não está instalado."
    echo "Instale uma versão compatível do Go e execute novamente."
    exit 1
fi

echo "[4/9] Verificando Piper..."

if command -v piper >/dev/null 2>&1; then
    echo "Piper encontrado:"
    command -v piper
else
    echo "AVISO: Piper não encontrado no PATH."
    echo "O binário deve estar disponível como /usr/local/bin/piper."
fi

echo "[5/9] Criando diretórios..."

mkdir -p \
    "$APP_DIR" \
    "$APP_DIR/data" \
    "$APP_DIR/audio" \
    "$APP_DIR/outputs" \
    "$APP_DIR/voices"

echo "[6/9] Verificando modelos de voz..."

MODEL_COUNT=0

if [ -d "$APP_DIR/voices" ]; then
    MODEL_COUNT=$(find "$APP_DIR/voices" -type f -name "*.onnx" | wc -l)
fi

if [ "$MODEL_COUNT" -eq 0 ]; then
    echo ""
    echo "AVISO: nenhum modelo Piper foi encontrado em:"
    echo "$APP_DIR/voices"
    echo ""
    echo "O código será compilado, mas o /tts não funcionará"
    echo "até que os modelos de voz estejam disponíveis."
else
    echo "Modelos encontrados: $MODEL_COUNT"
fi

echo "[7/9] Compilando SKTTS-Go..."

cd "$API_DIR"

go mod download
go build -o "$BIN" .

chmod +x "$BIN"

echo "Binário criado:"
ls -lh "$BIN"

echo "[8/9] Configurando systemd..."

cat > "$SERVICE" <<'SERVICE_EOF'
[Unit]
Description=SKTTS-Go Audio Engine
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/xtts
ExecStart=/opt/xtts/sktts-go
Restart=always
RestartSec=3

# Configure these variables before production:
Environment="XTTS_API_KEY="
Environment="SKTTS_ADMIN_KEY="
Environment="SKTTS_LINKEDIN_FREE_KEY="

Environment="SKTTS_DB=/opt/xtts/data/sktss.db"
Environment="SKTTS_ADDR=:8000"

[Install]
WantedBy=multi-user.target
SERVICE_EOF

systemctl daemon-reload
systemctl enable sktts-go

echo "[9/9] Iniciando serviço..."

systemctl restart sktts-go

sleep 2

if systemctl is-active --quiet sktts-go; then
    echo ""
    echo "=========================================="
    echo "       SKTTS-Go INSTALADO"
    echo "=========================================="
    echo ""
    echo "Serviço: ATIVO"
    echo "Binário: $BIN"
    echo "Porta:   8000"
    echo ""
else
    echo ""
    echo "ERRO: o serviço não iniciou."
    echo ""
    systemctl status sktts-go --no-pager
    echo ""
    echo "Logs:"
    journalctl -u sktts-go -n 50 --no-pager
    exit 1
fi

echo "Teste de saúde:"

if curl -fsS http://127.0.0.1:8000/health; then
    echo ""
    echo ""
    echo "=========================================="
    echo "             TUDO OK"
    echo "=========================================="
else
    echo ""
    echo "AVISO: serviço ativo, mas /health não respondeu."
    echo "Verifique:"
    echo "journalctl -u sktts-go -n 50 --no-pager"
fi

echo ""
echo "Próximos passos:"
echo ""
echo "1. Configure as chaves em:"
echo "$SERVICE"
echo ""
echo "2. Garanta que os modelos Piper estejam em:"
echo "$APP_DIR/voices/"
echo ""
echo "3. Reinicie:"
echo "systemctl restart sktts-go"
echo ""
echo "4. Verifique:"
echo "systemctl status sktts-go"
echo ""
echo "5. API:"
echo "http://SEU_IP:8000"
echo ""
