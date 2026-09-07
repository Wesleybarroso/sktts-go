#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="/opt/xtts/voices"
BASE_URL="https://huggingface.co/rhasspy/piper-voices/resolve/main"

echo "=========================================="
echo "       SKTTS-Go - Piper Voices"
echo "=========================================="

mkdir -p "$BASE_DIR"

download_voice() {
    local voice="$1"
    local quality="$2"
    local model="$3"

    local dir="$BASE_DIR/$voice"
    local prefix="pt_BR-${model}-${quality}"

    mkdir -p "$dir"

    echo ""
    echo "------------------------------------------"
    echo "Baixando voz: $voice"
    echo "Modelo: $prefix"
    echo "------------------------------------------"

    curl -fL --retry 3 --retry-delay 2 \
        -o "$dir/model.onnx" \
        "$BASE_URL/pt/pt_BR/$model/$quality/$prefix.onnx"

    curl -fL --retry 3 --retry-delay 2 \
        -o "$dir/model.onnx.json" \
        "$BASE_URL/pt/pt_BR/$model/$quality/$prefix.onnx.json"

    test -s "$dir/model.onnx"
    test -s "$dir/model.onnx.json"

    echo "OK: $voice"
}

download_voice "Cadu" "medium" "cadu"
download_voice "Edresson" "low" "edresson"
download_voice "Faber" "medium" "faber"
download_voice "Jeff" "medium" "jeff"

echo ""
echo "=========================================="
echo "       VOZES INSTALADAS"
echo "=========================================="

find "$BASE_DIR" -maxdepth 2 -type f \
    \( -name "model.onnx" -o -name "model.onnx.json" \) \
    -printf '%p\n' | sort

echo ""
echo "Tamanho:"
du -sh "$BASE_DIR"

echo ""
echo "Concluído."
