# Manual Técnico - Proyecto3
**Luciano Isaac Xiquín Ajpop - 201800632**
## Como instalar
Esta guía despliega el sistema completo en una máquina Windows con Kubernetes local. Todos los comandos son de **PowerShell** y se ejecutan desde la raíz de la carpeta `proyecto3`, salvo que se indique otra cosa.

| Herramienta | Uso | Instalación |
|---|---|---|
| Docker Desktop (backend WSL2) | contenedores, Zot y nodos de k3d | instalador oficial; dejar Kubernetes de Docker Desktop **desactivado** |
| Git | repositorio | `winget install Git.Git` |
| kubectl | cliente de Kubernetes | `winget install Kubernetes.kubectl` |
| Helm | charts de terceros | `winget install Helm.Helm` |
| k3d | clúster local | `choco install k3d`, `scoop install k3d` o binario de GitHub Releases |
| Python 3.12 + Locust | pruebas de carga | `winget install Python.Python.3.12` y `pip install locust` |

### ZOT

```powershell
New-Item -ItemType Directory -Force zot\certs, zot\data | Out-Null
cd zot

docker run --rm -v "${PWD}\certs:/certs" alpine/openssl req -x509 -newkey rsa:4096 -nodes -days 365 `
  -keyout /certs/zot.key -out /certs/zot.crt -subj "/CN=zot.local" `
  -addext "subjectAltName=DNS:zot.local,DNS:localhost,IP:127.0.0.1"

@'
{
  "distSpecVersion": "1.1.0",
  "storage": { "rootDirectory": "/var/lib/registry" },
  "http": { "address": "0.0.0.0", "port": "5000",
            "tls": { "cert": "/certs/zot.crt", "key": "/certs/zot.key" } },
  "log": { "level": "info" }
}
'@ | Set-Content -Encoding ascii config.json

docker network create munk8s
docker run -d --name zot --network munk8s --network-alias zot.local `
  -p 5000:5000 `
  -v "${PWD}\config.json:/etc/zot/config.json" `
  -v "${PWD}\certs:/certs" -v "${PWD}\data:/var/lib/registry" `
  ghcr.io/project-zot/zot-linux-amd64:latest
docker update --restart unless-stopped zot
cd ..
```

Verifica que responde por HTTPS (debe devolver `{}`):

```powershell
curl.exe -k https://localhost:5000/v2/
```


### Clúster k3d

El clúster se une a la red Docker de Zot para resolver `zot.local`. Se desactiva Traefik porque el Gateway lo provee Envoy.

```powershell
@'
configs:
  "zot.local:5000":
    tls:
      insecure_skip_verify: true
'@ | Set-Content -Encoding ascii registries.yaml

k3d cluster create munk8s --network munk8s --servers 1 --agents 2 `
  --k3s-arg "--disable=traefik@server:0" `
  --registry-config registries.yaml `
  -p "8080:80@loadbalancer"

kubectl get nodes
```

Prueba que el clúster descarga imágenes de Zot:

```powershell
docker pull busybox
docker tag busybox localhost:5000/test/busybox:1
docker push localhost:5000/test/busybox:1
kubectl run t --image=zot.local:5000/test/busybox:1 -- sleep 3600
kubectl get pod t        # debe llegar a Running
kubectl delete pod t
```

### Envoy Gateway (Gateway API)

```powershell
helm show chart oci://docker.io/envoyproxy/gateway-helm | Select-String "^version"
helm install eg oci://docker.io/envoyproxy/gateway-helm `
  --version <VERSION> -n envoy-gateway-system --create-namespace
kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
```

El chart instala los CRDs estándar de Gateway API (`GatewayClass`, `Gateway`, `HTTPRoute`).

### 4.6 Construir y publicar las imágenes

```powershell
$REG = "localhost:5000/munk8s"

# Imágenes propias
docker build -t "$REG/rust-api:v2"          rust-api
docker build -t "$REG/go-api:v1"            -f go-deployment1/Dockerfile.api        go-deployment1
docker build -t "$REG/go-grpc-client:v2"    -f go-deployment1/Dockerfile.grpcclient go-deployment1
docker build -t "$REG/go-grpc-server:v1"    -f go-deployment2/Dockerfile.server     go-deployment2
docker build -t "$REG/go-rabbit-writer:v1"  -f go-deployment2/Dockerfile.writer     go-deployment2
docker build -t "$REG/go-consumer:v1"       go-consumer
docker build -t "$REG/munk8s-operator:v2"   operator
docker build -t "$REG/grafana-valkey:v1"    grafana
docker build -t "$REG/dapr-publisher:v1"    -f go-dapr/Dockerfile.publisher  go-dapr
docker build -t "$REG/dapr-subscriber:v1"   -f go-dapr/Dockerfile.subscriber go-dapr

# Imágenes de terceros, espejadas en Zot
docker pull rabbitmq:3.13-management
docker tag  rabbitmq:3.13-management "$REG/rabbitmq:3.13-management"
docker pull valkey/valkey:8
docker tag  valkey/valkey:8 "$REG/valkey:8"

# Publicar todo
$imgs = "rust-api:v2","go-api:v1","go-grpc-client:v2","go-grpc-server:v1","go-rabbit-writer:v1",
        "go-consumer:v1","munk8s-operator:v2","grafana-valkey:v1","dapr-publisher:v1",
        "dapr-subscriber:v1","rabbitmq:3.13-management","valkey:8"
$imgs | ForEach-Object { docker push "$REG/$_" }
```

Comprueba el catálogo:

```powershell
curl.exe -k https://localhost:5000/v2/_catalog
```

### 4.7 Despliegue del sistema

El orden importa: RabbitMQ y el Operator deben existir antes de los servicios que dependen de ellos.

**a) Namespace y RabbitMQ**

```powershell
kubectl apply -f k8s/00-namespace.yaml
kubectl apply -f k8s/40-rabbitmq.yaml
kubectl -n munk8s rollout status statefulset/rabbitmq --timeout=180s
```

**b) Operator y Valkey**

```powershell
kubectl apply -f operator/deploy/crd.yaml
kubectl apply -f operator/deploy/crd-grafana.yaml
kubectl apply -f operator/deploy/rbac.yaml
kubectl apply -f operator/deploy/manager.yaml
kubectl -n munk8s-operator rollout status deploy/munk8s-operator

kubectl apply -f operator/deploy/sample-valkeyinstance.yaml
kubectl -n munk8s wait --for=condition=Ready pod/valkey-0 --timeout=180s
```

**c) Servicios de la aplicación y Gateway**

```powershell
kubectl apply -f k8s/50-go-deployment2.yaml
kubectl apply -f k8s/20-go-deployment1.yaml
kubectl apply -f k8s/10-rust-api.yaml
kubectl apply -f k8s/30-gateway.yaml
kubectl apply -f k8s/60-consumer.yaml
```

**d) Grafana (Custom Resource)**

```powershell
kubectl apply -f operator/deploy/sample-grafanainstance.yaml
kubectl -n munk8s rollout status deploy/grafana
```

## Arquitectura del Proyecto
Locust > Gateway > Rust > Go 1 > Go 2 > RabbitMQ <> Consumer > Valkey (El operator los crea) > Grafana

El sistema recibe reportes militares simulados y los procesa en varias etapas dentro de un clúster de Kubernetes. Locust genera los reportes en formato JSON y los envía al Gateway API, que los dirige hacia la API REST en Rust. Rust valida que el país y los rangos sean correctos y reenvía el reporte al primer servicio en Go. Ese servicio lo convierte a un mensaje gRPC y lo envía al segundo servicio en Go, el cual lo publica en una cola de RabbitMQ. Un Consumer en Go lee los mensajes de la cola, calcula los máximos, mínimos, el top de países, la moda y los contadores, y guarda los resultados en Valkey. Grafana lee esos datos desde Valkey y los muestra en un dashboard. Una ruta alternativa, /dapr-201800632, hace el mismo recorrido usando Dapr en lugar de gRPC, para poder comparar ambos enfoques.

Esta organización permite que cada componente se escale y se recupere por separado. La cola de RabbitMQ guarda los reportes cuando llegan más rápido de lo que el Consumer los procesa, así que no se pierden. El HPA crea más réplicas de Rust cuando el uso de CPU supera el 30%, y los servicios en Go se pueden probar con 1 y 2 réplicas para medir el rendimiento. Valkey y Grafana se crean y se mantienen mediante un Operator, a partir de Custom Resources declarados en el clúster, y Valkey guarda sus datos en un volumen persistente que sobrevive a los reinicios. Todas las imágenes de los componentes se publican en Zot, un registro de containers con HTTPS fuera del clúster, y desde ahí las descarga Kubernetes.        

## Pruebas
### Fase 2
#### Con 2 replicas
![Imagen 1](imgs/1.png)
![Imagen 1](imgs/2.png)
![Imagen 1](imgs/3.png)
![Imagen 1](imgs/4.png)
![Imagen 1](imgs/5.png)
----
#### Con una replica
![Imagen 1](imgs/6.png)
![Imagen 1](imgs/7.png)
![Imagen 1](imgs/8.png)
![Imagen 1](imgs/9.png)
![Imagen 1](imgs/10.png)
----
### Fase 3  
#### Configuración de Valkey y el Operator
![Imagen 11](imgs/11.png)
![Imagen 12](imgs/12.png)
![Imagen 13](imgs/13.png)
#### Datos de Valkey
![Valores de Valkey](imgs/valkey.png)
* Contadores
* Máximos y mínimos
* Top de países
* Moda
* Serie temporal
* Cola vacía (Consumer encendido)

#### Persistencia
Luego de borrar el pod de Valkey los datos se mantuvieron
```powershell
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3> kubectl -n munk8s exec valkey-0 -- valkey-cli GET count:total 
63708
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3> kubectl -n munk8s get pvc 
NAME              STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   VOLUMEATTRIBUTESCLASS   AGE
data-rabbitmq-0   Bound    pvc-1157109b-9129-49a6-b81c-6144d67c56da   1Gi        RWO            local-path     <unset>                 4d20h
data-valkey-0     Bound    pvc-b389d2ce-88c0-47f7-87c2-f0f92a9050b9   1Gi        RWO            local-path     <unset>                 4h24m
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3> kubectl -n munk8s delete pod valkey-0
pod "valkey-0" deleted from munk8s namespace
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3> kubectl -n munk8s wait --for=condition=Ready pod/valkey-0 --timeout=120s
pod/valkey-0 condition met
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3> kubectl -n munk8s exec valkey-0 -- valkey-cli GET count:total
63708
PS C:\Users\lucia\Documents\GitHub\201800632_LAB_SO1_2S2026\Proyecto3>
``` 

#### Reconciliación del Operator
Un Operator no solo crea recursos: mantiene el estado deseado.
* A
![Comandos](imgs/A.png)
El Operator detecta que falta un recurso de su release y lo recrea. Debe reaparecer en segundos o, a lo sumo, en el siguiente ciclo de reconciliación (alrededor de un minuto; no lo pude medir). Captura el -w mostrando la desaparición y la reaparición, y el log del Operator.

* B
![Comandos](imgs/B.png)
Un cambio en el Custom Resource se propaga

* C
![Comandos](imgs/C.png)
Desaparecen el StatefulSet y el Service, porque el Operator desinstala la release. El PVC se queda. Después se restaura.
El valor es el mismo que antes de borrar. Los datos persisten.

### Fase 4
#### Reconciliacion del Custom Resource
![Comandos](imgs/21.png)
#### Valkey se conecta a Grafana
![Valkey y Grafana](imgs/22.png)
