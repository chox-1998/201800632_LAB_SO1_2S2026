// RabbitMQ Consumer: lee war_reports, valida y almacena agregados en Valkey.
// Concurrencia: N workers sobre un mismo channel con prefetch y ack manual.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

const queueName = "war_reports"

// Script atomico: todos los agregados se actualizan en una sola operacion,
// por lo que funciona correctamente con varias replicas del consumer.
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

func (r report) valid() bool {
	return validCountries[r.Country] &&
		r.WarplanesInAir >= 0 && r.WarplanesInAir <= 50 &&
		r.WarshipsInWater >= 0 && r.WarshipsInWater <= 30
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
	workers, _ := strconv.Atoi(getenv("WORKERS", "8"))
	prefetch, _ := strconv.Atoi(getenv("PREFETCH", "50"))

	rdb := redis.NewClient(&redis.Options{
		Addr:     getenv("VALKEY_ADDR", "valkey:6379"),
		PoolSize: workers * 2,
	})
	for i := 0; ; i++ {
		if err := rdb.Ping(context.Background()).Err(); err == nil {
			break
		} else if i >= 60 {
			log.Fatalf("valkey no disponible: %v", err)
		} else {
			log.Printf("esperando Valkey (%d/60): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
	}
	script := redis.NewScript(luaScript)

	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:5672/",
		url.QueryEscape(getenv("RABBITMQ_USER", "guest")),
		url.QueryEscape(getenv("RABBITMQ_PASS", "guest")),
		getenv("RABBITMQ_HOST", "rabbitmq"))
	var conn *amqp.Connection
	var err error
	for i := 0; i < 30; i++ {
		if conn, err = amqp.Dial(amqpURL); err == nil {
			break
		}
		log.Printf("esperando RabbitMQ (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	if _, err = ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		log.Fatal(err)
	}
	if err = ch.Qos(prefetch, 0, false); err != nil {
		log.Fatal(err)
	}
	host, _ := os.Hostname()
	msgs, err := ch.Consume(queueName, "consumer-"+host, false, false, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	var processed, failed, dropped atomic.Int64
	for i := 0; i < workers; i++ {
		go func() {
			for d := range msgs {
				var r report
				if json.Unmarshal(d.Body, &r) != nil || !r.valid() {
					dropped.Add(1)
					d.Reject(false) // mensaje invalido: no reencolar
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				err := script.Run(ctx, rdb, nil, r.Country, r.WarplanesInAir, r.WarshipsInWater).Err()
				cancel()
				if err != nil {
					failed.Add(1)
					log.Printf("valkey: %v (reencolando)", err)
					d.Nack(false, true)
					time.Sleep(200 * time.Millisecond)
					continue
				}
				d.Ack(false)
				processed.Add(1)
			}
		}()
	}

	go func() {
		for range time.Tick(10 * time.Second) {
			log.Printf("stats: procesados=%d fallidos=%d descartados=%d",
				processed.Load(), failed.Load(), dropped.Load())
		}
	}()

	log.Printf("consumer listo: workers=%d prefetch=%d", workers, prefetch)
	e := <-conn.NotifyClose(make(chan *amqp.Error, 1))
	log.Fatalf("conexion AMQP cerrada: %v", e)
}
