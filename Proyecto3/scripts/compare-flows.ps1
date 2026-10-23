<#
 Compara el flujo gRPC (/grpc-201800632) contra Dapr (/dapr-201800632)
 con la misma carga. Uso: .\scripts\compare-flows.ps1 -Users 100 -Rate 10 -Duration 2m
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

$summary = @()
foreach ($r in @("grpc", "dapr")) {
  Write-Host "=== Flujo $r ===" -ForegroundColor Cyan
  $env:ROUTE = $r
  $before = Get-Total
  locust -f locust/locustfile.py --headless -u $Users -r $Rate -t $Duration --csv "results/flow-$r" --only-summary

  # Esperar a que el total deje de crecer (flujo asincrono), max 180 s
  $t0 = Get-Date; $last = -1; $stable = 0
  while ($stable -lt 3 -and ((Get-Date) - $t0).TotalSeconds -lt 180) {
    Start-Sleep 3
    $now = Get-Total
    if ($now -eq $last) { $stable++ } else { $stable = 0; $last = $now }
  }
  $settle = [int]((Get-Date) - $t0).TotalSeconds
  $after = Get-Total

  $row = Import-Csv "results/flow-${r}_stats.csv" | Where-Object { $_.Name -eq "Aggregated" }
  $summary += [pscustomobject]@{
    Flow = $r; Requests = $row.'Request Count'; Failures = $row.'Failure Count'
    RPS = $row.'Requests/s'; P50_ms = $row.'50%'; P95_ms = $row.'95%'; P99_ms = $row.'99%'
    ValkeyDelta = $after - $before; SettleSeconds = $settle
  }
}
$summary | Format-Table -AutoSize
$summary | Export-Csv results/flows-summary.csv -NoTypeInformation
