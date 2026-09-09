#!/bin/bash
# containers.sh — Crea 5 contenedores Docker aleatoriamente
# Ejecutado por el cronjob cada 2 minutos

echo "[containers] Lanzando 5 contenedores aleatorios..."

# Las 3 opciones de imagen según el spec
launch_container() {
    local choice=$((RANDOM % 3))
    case $choice in
        0)
            # Alto consumo RAM — go-client
            docker run -d --rm \
                --label "tipo=alto" \
                roldyoran/go-client
            echo "[containers] Lanzado: go-client (alto consumo RAM)"
            ;;
        1)
            # Alto consumo CPU — alpine con bc
            docker run -d --rm \
                --label "tipo=alto" \
                alpine sh -c "while true; do echo '2^20' | bc > /dev/null; sleep 2; done"
            echo "[containers] Lanzado: alpine CPU stress (alto consumo CPU)"
            ;;
        2)
            # Bajo consumo — alpine sleep
            docker run -d --rm \
                --label "tipo=bajo" \
                alpine sleep 240
            echo "[containers] Lanzado: alpine sleep (bajo consumo)"
            ;;
    esac
}

for i in $(seq 1 5); do
    launch_container
done

echo "[containers] Listo — 5 contenedores creados."
