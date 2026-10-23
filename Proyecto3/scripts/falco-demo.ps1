<#
 Simula la amenaza: entra a un pod y lee /etc/shadow y /app/secretos,
 luego muestra la alerta del Falco que corre en el MISMO nodo.
 Uso: .\scripts\falco-demo.ps1            (pod de rust-api)
      .\scripts\falco-demo.ps1 -App go-deployment1
#>
param([string]$App = "rust-api")
$ns = "munk8s"

$pod  = kubectl -n $ns get pod -l app=$App -o jsonpath="{.items[0].metadata.name}"
$node = kubectl -n $ns get pod $pod -o jsonpath="{.spec.nodeName}"
$fpod = kubectl -n falco get pod -l app.kubernetes.io/name=falco `
          --field-selector "spec.nodeName=$node" -o jsonpath="{.items[0].metadata.name}"
Write-Host "Pod objetivo: $pod  | Nodo: $node  | Falco: $fpod" -ForegroundColor Cyan

Write-Host "`n--- Simulando amenaza ---" -ForegroundColor Yellow
kubectl -n $ns exec $pod -- sh -c "mkdir -p /app/secretos; cat /etc/shadow; ls /app/secretos"
Start-Sleep -Seconds 3

Write-Host "`n--- Alertas de Falco ---" -ForegroundColor Yellow
kubectl -n falco logs $fpod -c falco --tail=300 | Select-String "MUNK8S|sensible"
