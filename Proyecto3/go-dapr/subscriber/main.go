// Dapr Subscriber: se suscribe al topic (via SDK) y guarda en Valkey
// con el mismo script atomico que usa el consumer del flujo gRPC.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/redis/go-redis/v9"
)

const luaScript = `
local c, p, s = ARGV[1], tonumber(ARGV[2]), tonumber(ARGV[3])
local function setmax(k, v) local cur = redis.call('GET', k); if (not cur) or v > tonumber(cur) then redis.call('SET', k, v) end end
local function setmin(k, v) local cur = redis.call('GET', k); if (not cur) or v < tonumber(cur) then redis.call('SET', k, v) end end
setmax('max:planes', p); setmin('min:planes', p)
setmax('max:ships',  s); setmin('min:ships',  s)
redis.call('ZINCRBY', 'top:planes', p, c)
redis.call('ZINCRBY', 'top:ships',  s, c)
redis.call('ZINCRBY', 'freq:planes', 1, p)
redis.call('ZINCRBY', 'freq:ships',  1, s)
redis.call('SET', 'mode:planes', redis.call('ZREVRANGE', 'freq:planes', 0, 0)[1])
redis.call('SET', 'mode:ships',  redis.call('ZREVRANGE', 'freq:ships',  0, 0)[1])
redis.call('INCR', 'count:' .. c)
redis.call('INCR', 'count:total')
redis.call('XADD', 'ts:' .. c, 'MAXLEN', '~', 100000, '*', 'planes', p, 'ships', s)
return 1
`

var validCountries = map[string]bool{"USA": true, "RUS": true, "CHN": true, "ESP": true, "GTM": true}

type report struct {
	Country         string `json:"country"`
	WarplanesInAir  int    `json:"warplanes_in_air"`
	WarshipsInWater int    `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: getenv("VALKEY_ADDR", "valkey:6379"), PoolSize: 16})
	for i := 0; ; i++ {
		err := rdb.Ping(context.Background()).Err()
		if err == nil {
			break
		}
		if i >= 60 {
			log.Fatalf("valkey no disponible: %v", err)
		}
		log.Printf("esperando Valkey (%d/60): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	script := redis.NewScript(luaScript)

	handler := func(ctx context.Context, e *common.TopicEvent) (bool, error) {
		var raw []byte
		switch d := e.Data.(type) {
		case []byte:
			raw = d
		case string:
			raw = []byte(d)
		default:
			raw, _ = json.Marshal(d)
		}
		if len(raw) == 0 {
			raw = e.RawData
		}
		var r report
		if json.Unmarshal(raw, &r) != nil || !validCountries[r.Country] ||
			r.WarplanesInAir < 0 || r.WarplanesInAir > 50 ||
			r.WarshipsInWater < 0 || r.WarshipsInWater > 30 {
			log.Printf("evento invalido descartado: %s", string(raw))
			return false, nil // no reintentar
		}
		if err := script.Run(ctx, rdb, nil, r.Country, r.WarplanesInAir, r.WarshipsInWater).Err(); err != nil {
			log.Printf("valkey: %v (reintento)", err)
			return true, err // Dapr reentrega
		}
		return false, nil
	}

	s := daprd.NewService(":8091")
	sub := &common.Subscription{PubsubName: "rabbitmq-pubsub", Topic: "war-reports", Route: "/events"}
	if err := s.AddTopicEventHandler(sub, handler); err != nil {
		log.Fatalf("subscription: %v", err)
	}
	log.Println("dapr subscriber escuchando en :8091")
	if err := s.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("service: %v", err)
	}
}
