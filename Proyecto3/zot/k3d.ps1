@'
configs:
  "zot.local:5000":
    tls:
      insecure_skip_verify: true
'@ | Set-Content -Encoding ascii registries.yaml # Aquí se crea el archivo de configuración para el registro de contenedores ZOT, que se va a usar en K3D

k3d cluster create munk8s --network munk8s --servers 1 --agents 2 ` --k3s-arg "--disable=traefik@server:0" ` --registry-config registries.yaml ` -p "8080:80@loadbalancer" # Y aquí se crel el cluster K3D; 1 Server; 2 Agentes; k3s-args es para deshabilitar Traefik; registry-config es para usar el registro de contenedores ZOT; -p es para exponer el puerto 8080 del cluster K3D al puerto 80 del host
  # Traefik es un proxy inverso y balanceador de carga que se utiliza para enrutar el tráfico hacia los servicios dentro del cluster K3D. Al deshabilitar Traefik, se evita que este maneje el tráfico entrante, lo que puede ser útil si se desea utilizar otro método de enrutamiento o balanceo de carga.