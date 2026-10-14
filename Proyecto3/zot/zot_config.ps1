New-Item -ItemType Directory -Force certs, data | Out-Null

docker run --rm -v "${PWD}\certs:/certs" alpine/openssl req -x509 -newkey rsa:4096 -nodes -days 365 -keyout /certs/zot.key -out /certs/zot.crt -subj "/CN=zot.local" -addext "subjectAltName=DNS:zot.local,DNS:localhost,IP:127.0.0.1"
#docker run --rm -v "${PWD}\certs:/certs" alpine/openssl req -x509 -newkey rsa:4096 -nodes -days 365 ` 
#  -keyout /certs/zot.key -out /certs/zot.crt -subj "/CN=zot.local" `
#  -addext "subjectAltName=DNS:zot.local,DNS:localhost,IP:127.0.0.1"
@'
{
  "distSpecVersion": "1.1.0",
  "storage": { "rootDirectory": "/var/lib/registry" },
  "http": { "address": "0.0.0.0", "port": "5000",
            "tls": { "cert": "/certs/zot.crt", "key": "/certs/zot.key" } },
  "log": { "level": "info" }
}
'@ | Set-Content -Encoding ascii config.json # Crear el archivo de configuración para ZOT

docker network create munk8s # Qué es munk8s? Es la red que se va a usar para el registro de contenedores ZOT
docker run -d --name zot --network munk8s --network-alias zot.local -p 5000:5000 -v "${PWD}\config.json:/etc/zot/config.json" -v "${PWD}\certs:/certs" -v "${PWD}\data:/var/lib/registry" ghcr.io/project-zot/zot-linux-amd64:latest

  # En Zot se hace push de los contenedores construidos
  # Apenas si entiendo esto
  # Para correr el archivo ps1 en PowerShell, se debe ejecutar el comando: .\zot.ps1