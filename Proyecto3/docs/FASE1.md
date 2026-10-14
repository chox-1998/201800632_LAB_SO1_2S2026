# Fase 1 - orden de ejecucion

1. `.\zot.ps1`


2. Dentro del directorio de zot:
`./scripts/build-push.sh v1`   -> construye y sube 3 imagenes a Zot
3. `./scripts/deploy.sh`          -> namespace, deployments, HPA, gateway
## Instalar Envoy Gateway
```Powershell
helm install eg oci://docker.io/envoyproxy/gateway-helm `
  --version 1.9.2 -n envoy-gateway-system --create-namespace

kubectl wait --timeout=5m -n envoy-gateway-system deployment/envoy-gateway --for=condition=Available
```


4. Prueba manual:

       curl -i -X POST http://localhost:8080/grpc-201800632 \
         -H 'Content-Type: application/json' \
         -d '{"country":"RUS","warplanes_in_air":42,"warships_in_water":14,"timestamp":"2026-03-12T20:15:30Z"}'

5. Logs del salto final (modo stub):

       kubectl -n munk8s logs deploy/go-deployment1 -c grpc-client

6. Locust (UI en http://localhost:8089):

       cd locust && python -m locust -f locustfile.py


## Pausar
```Powershell
k3d cluster stop munk8s
docker stop zot
```

## Reanudar
```Powershell
docker start zot
k3d cluster start munk8s
kubectl -n munk8s get pods -w
```


**Nota: Verificar los datos de Locust**