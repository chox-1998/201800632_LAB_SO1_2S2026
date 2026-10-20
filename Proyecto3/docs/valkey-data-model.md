# Modelo de datos en Valkey

| Clave | Tipo | Contenido | Panel de Grafana |
|---|---|---|---|
| `max:planes`, `min:planes` | string | max/min global de aviones | Max/Min Aviones |
| `max:ships`, `min:ships` | string | max/min global de barcos | Max/Min Barcos |
| `top:planes`, `top:ships` | sorted set | pais -> suma acumulada | Top paises |
| `freq:planes`, `freq:ships` | sorted set | valor -> veces visto | (base de la moda) |
| `mode:planes`, `mode:ships` | string | valor mas frecuente | Moda |
| `count:<PAIS>`, `count:total` | string (int) | reportes recibidos | Reportes del pais |
| `ts:<PAIS>` | stream (MAXLEN ~100000) | entradas `planes`, `ships` | Serie temporal |

Todo se actualiza en un unico script Lua (atomico), por lo que funciona con N consumers.
Los streams se recortan con MAXLEN para acotar la memoria (politica de expiracion).
