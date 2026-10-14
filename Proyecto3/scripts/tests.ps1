$body = '{"country":"RUS","warplanes_in_air":42,"warships_in_water":14,"timestamp":"2026-03-12T20:15:30Z"}' # Body para mandar
Invoke-RestMethod -Method Post -Uri http://localhost:8080/grpc-201800632 -ContentType "application/json" -Body $body

kubectl -n munk8s logs deploy/go-deployment1 -c grpc-client