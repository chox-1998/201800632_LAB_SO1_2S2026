#!/bin/bash

sudo apt update
sudo apt install -y build-essential linux-headers-$(uname -r) golang docker.io docker-compose
sudo usermod -aG docker $USER  

echo "Dependencias instaladas. Reiniciar sesión"
