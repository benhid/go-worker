package main

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	serverPort   string
	warmPoolSize int
)

func init() {
	flag.StringVar(&serverPort, "listen", ":8090", "[ip]:port to listen on for HTTP.")
	flag.StringVar(&agentImage, "image", "hello-python", "Docker image to run workloads.")
	flag.IntVar(&warmPoolSize, "min-size", 10, "The size of the warm pool.")
	// TODO - Add instance-reuse-policy
	prometheus.MustRegister(receivedMetric)
	prometheus.MustRegister(errorsMetric)
	prometheus.MustRegister(lastRequestMetric)
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("Warm pool size: %d", warmPoolSize)

	WarmContainers := make(chan runningContainer, warmPoolSize)
	defer close(WarmContainers)

	go fillPool(ctx, WarmContainers)

	// Start HTTP server to handle job requests.
	http.HandleFunc("/_/health", makeHealthHandler())
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/run", makeRunHandler(ctx, WarmContainers))

	log.Info("Starting server at port " + serverPort)

	log.Fatal(http.ListenAndServe(serverPort, nil))
}

func makeHealthHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Worker is running"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func makeRunHandler(ctx context.Context, WarmContainers <-chan runningContainer) func(
	http.ResponseWriter, *http.Request,
) {
	return func(w http.ResponseWriter, r *http.Request) {
		lastRequestMetric.SetToCurrentTime()

		if r.Method != "POST" {
			errorsMetric.With(prometheus.Labels{"type": "wrong_method"}).Inc()
			log.Error("Received wrong method")
			http.Error(w, "Expected request to be POSTed", http.StatusMethodNotAllowed)
			return
		}

		job, err := readRequestBody(r)
		if err != nil {
			errorsMetric.With(prometheus.Labels{"type": "decode"}).Inc()
			log.WithError(err).Error("Received invalid request")
			http.Error(w, "Decode failed", http.StatusInternalServerError)
			return
		}

		receivedMetric.Inc()

		err = job.Validate()
		if err != nil {
			log.WithError(err).Error("Received invalid job")
			http.Error(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		}

		go job.run(ctx, WarmContainers)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}
}

// readRequestBody reads the body of the request and unmarshals it.
func readRequestBody(r *http.Request) (Job, error) {
	defer r.Body.Close()

	job := Job{}
	err := json.NewDecoder(r.Body).Decode(&job)

	return job, err
}
