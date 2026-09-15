#!/bin/bash
# load_module.sh — Carga el módulo de kernel continfo
# Ejecutado por el Daemon de Go al iniciar

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$SCRIPT_DIR/../module-kernels"
MODULE_NAME="continfo"
PROC_FILE="/proc/continfo_pr2_so1_201800632"   

echo "[load_module] Verificando módulo de kernel..."

if lsmod | grep -q "^${MODULE_NAME} "; then
    echo "[load_module] El módulo ya está cargado."
    exit 0
fi

if [ ! -f "$MODULE_DIR/${MODULE_NAME}.ko" ]; then
    echo "[load_module] Compilando módulo..."
    make -C "$MODULE_DIR" all
fi

# Cargar módulo
echo "[load_module] Cargando módulo..."
sudo insmod "$MODULE_DIR/${MODULE_NAME}.ko"

# Verificar que /proc se creó
if [ -f "$PROC_FILE" ]; then
    echo "[load_module] OK — $PROC_FILE disponible."
else
    echo "[load_module] ERROR — $PROC_FILE no se creó."
    exit 1
fi
