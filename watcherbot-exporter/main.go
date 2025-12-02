package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "watcherbot_requests_total",
	Help: "This is to count the total requests that are to come up ",
})

var requestLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: "monitoring",
	Name:      "request_latency_seconds",
	Help:      "Latency Percentiles in Histogram Buckets",
	Buckets:   []float64{0.1, 0.25, 0.5},
})

var activeTasks = prometheus.NewGauge(prometheus.GaugeOpts{
	Name:      "active_tasks",
	Namespace: "watcherBot",
	Help:      "Current Number of tasks being processed",
})

func init() {
	prometheus.MustRegister(requestCounter, requestLatency, activeTasks)
}

func hello(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	defer func() {
		requestLatency.Observe(time.Since(now).Seconds())
	}()

	requestCounter.Inc()
	fmt.Fprintf(w, "Request processed and counter incremented! \n", requestCounter)

}

func startTask(w http.ResponseWriter, r *http.Request) {
	activeTasks.Inc()
	fmt.Fprintf(w, "Request processed and active tasks incremented! \n", activeTasks)
}

func finishTask(w http.ResponseWriter, r *http.Request) {
	activeTasks.Dec()
	fmt.Fprintf(w, "Request processed and decremented task! \n", activeTasks)
}

func main() {
	http.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/start_task", startTask)
	http.HandleFunc("/finish_task", finishTask)

	port := ":8088"
	log.Printf("WatcherBot Exporter running on port %s...", port)
	log.Printf("Test endpoints: /hello, /start_task, /finish_task")

	log.Fatal(http.ListenAndServe(port, nil))
}
