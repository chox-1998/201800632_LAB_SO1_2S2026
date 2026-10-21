<#
 Compara configuraciones de replicas bajo la misma carga de Locust.
 Uso (desde la raiz de proyecto3):
   .\scripts\loadtest.ps1 -Users 100 -Rate 10 -Duration 2m
 Genera results\<config>_stats.csv y results\summary.csv
#>
param(
  [int]$Users = 100,
  [int]$Rate = 10,
  [string]$Duration = "2m",
  [string]$Target = "http://localhost:8080"
)
$ns = "munk8s"
New-Item -ItemType Directory -Force results | Out-Null
$env:TARGET_HOST = $Target

function Get-Total { [int](kubectl -n $ns exec valkey-0 -- valkey-cli GET count:total) }
function Get-QueueDepth {
  $line = kubectl -n $ns exec rabbitmq-0 -- rabbitmqctl list_queues --quiet --no-table-headers name messages |
          Where-Object { $_ -match '^war_reports' }
  if (-not $line) { return -1 }
  [int](($line -split '\s+')[1])
}

$configs = @(
  @{ writer = 1; consumer = 1 },
  @{ writer = 2; consumer = 1 },
  @{ writer = 1; consumer = 2 },
  @{ writer = 2; consumer = 2 }
)
$summary = @()

foreach ($c in $configs) {
  $tag = "w$($c.writer)-c$($c.consumer)"
  Write-Host "=== Configuracion $tag ===" -ForegroundColor Cyan
  kubectl -n $ns scale deploy/go-deployment2 --replicas=$($c.writer) | Out-Null
  kubectl -n $ns scale deploy/rabbit-consumer --replicas=$($c.consumer) | Out-Null
  kubectl -n $ns rollout status deploy/go-deployment2 | Out-Null
  kubectl -n $ns rollout status deploy/rabbit-consumer | Out-Null
  Write-Host "Esperando 45s para que el cliente gRPC redescubra los pods..."
  Start-Sleep -Seconds 45

  $before = Get-Total
  locust -f locust/locustfile.py --headless -u $Users -r $Rate -t $Duration --csv "results/$tag" --only-summary

  # Esperar a que el consumer vacie la cola (max 180 s)
  $t0 = Get-Date
  while ((Get-QueueDepth) -gt 0 -and ((Get-Date) - $t0).TotalSeconds -lt 180) { Start-Sleep 3 }
  $drain = [int]((Get-Date) - $t0).TotalSeconds
  $after = Get-Total

  $row = Import-Csv "results/${tag}_stats.csv" | Where-Object { $_.Name -eq "Aggregated" }
  $summary += [pscustomobject]@{
    Config        = $tag
    Requests      = $row.'Request Count'
    Failures      = $row.'Failure Count'
    RPS           = $row.'Requests/s'
    P50_ms        = $row.'50%'
    P95_ms        = $row.'95%'
    P99_ms        = $row.'99%'
    ValkeyDelta   = $after - $before
    DrainSeconds  = $drain
  }
}
$summary | Format-Table -AutoSize
$summary | Export-Csv results/summary.csv -NoTypeInformation
Write-Host "Listo: results\summary.csv"
