#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="/opt/xtts/voices"
BASE_URL="https://huggingface.co/rhasspy/piper-voices/resolve/main"

echo "=========================================="
echo "   SKTTS-Go - Download das vozes Piper"
echo "=========================================="

mkdir -p "$BASE_DIR"

download_voice() {
    local name="$1"
    local model="$2"
    local quality="$3"

    local dir="$BASE_DIR/$name"
    local prefix="pt_BR-${model}-${quality}"

    echo ""
    echo ">>> Instalando voz: $name"

    mkdir -p "$dir"

    curl -fL --retry 3 --retry-delay 2 \
        -o "$dir/model.onnx" \
        "$BASE_URL/pt/pt_BR/$model/$quality/$prefix.onnx"

    curl -fL --retry 3 --retry-delay 2 \
        -o "$dir/model.onnx.json" \
        "$BASE_URL/pt/pt_BR/$model/$quality/$prefix.onnx.json"

    test -s "$dir/model.onnx"
    test -s "$dir/model.onnx.json"

    echo "OK: $name"
}

# Vozes oficiais Piper / Português Brasileiro
download_voice "Cadu" "cadu" "medium"
download_voice "Edresson" "edresson" "low"
download_voice "Faber" "faber" "medium"
download_voice "Jeff" "jeff" "medium"

echo ""
echo "=========================================="
echo "       VOZES INSTALADAS"
echo "=========================================="

for voice in Cadu Edresson Faber Jeff; do
    if [ -f "$BASE_DIR/$voice/model.onnx" ] &&
       [ -f "$BASE_DIR/$voice/model.onnx.json" ]; then
        echo "✓ $voice"
    else
        echo "✗ $voice"
        exit 1
    fi
done

echo ""
echo "Arquivos:"
find "$BASE_DIR" -maxdepth 2 -type f \
    \( -name "model.onnx" -o -name "model.onnx.json" \) \
    -printf '%p\n' | sort

echo ""
echo "Tamanho total:"
du -sh "$BASE_DIR"

echo ""
echo "Download concluído."
