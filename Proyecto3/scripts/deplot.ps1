$REG = "localhost:5000/munk8s"; $TAG = "v1" # Esto define el registro de contenedores ZOT y la etiqueta que se va a usar para los contenedores construidos
# Las imagenes que se contruiran y se subiran al Zot
docker build -t "$REG/rust-api:$TAG" rust-api
docker build -t "$REG/go-api:$TAG" -f go-deployment1/Dockerfile.api go-deployment1
docker build -t "$REG/go-grpc-client:$TAG" -f go-deployment1/Dockerfile.grpcclient go-deployment1
"rust-api","go-api","go-grpc-client" | ForEach-Object { docker push "$REG/${_}:$TAG" }


kubectl apply -f k8s/00-namespace.yaml # Esto crea el namespace munk8s, que es donde se van a desplegar los servicios y los deployments
kubectl apply -f k8s/10-rust-api.yaml -f k8s/20-go-deployment1.yaml # Esto es para el Rust y el Go en el cluster
kubectl apply -f k8s/30-gateway.yaml # Gateway
kubectl -n munk8s get deploy,svc,hpa,gateway,httproute # y finalmentes esto es para ver los deployments, servicios, HPA, gateway y httproute en el namespace munk8s

# 