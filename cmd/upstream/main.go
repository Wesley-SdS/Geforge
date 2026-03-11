package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	port := flag.Int("port", 3001, "listen port")
	name := flag.String("name", "upstream-1", "server name")
	errorRate := flag.Float64("error-rate", 0.05, "fraction of requests that return 500")
	minLatency := flag.Int("min-latency", 10, "minimum response latency in ms")
	maxLatency := flag.Int("max-latency", 200, "maximum response latency in ms")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		latency := *minLatency + rand.Intn(*maxLatency-*minLatency+1)
		time.Sleep(time.Duration(latency) * time.Millisecond)

		if rand.Float64() < *errorRate {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":  "simulated upstream error",
				"server": *name,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"server":  *name,
			"path":    r.URL.Path,
			"method":  r.Method,
			"latency": fmt.Sprintf("%dms", latency),
			"headers": map[string]string{
				"X-Forwarded-For": r.Header.Get("X-Forwarded-For"),
				"X-Request-ID":    r.Header.Get("X-Request-ID"),
			},
		})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("%s listening on %s (error_rate=%.0f%%, latency=%d-%dms)\n",
		*name, addr, *errorRate*100, *minLatency, *maxLatency)
	log.Fatal(http.ListenAndServe(addr, mux))
}
