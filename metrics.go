package main

import "github.com/prometheus/client_golang/prometheus"

var (
	receivedMetric = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "agent",
			Subsystem: "http",
			Name:      "received_total",
			Help:      "Number of incoming jobs received.",
		})
	errorsMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "agent",
			Subsystem: "http",
			Name:      "errors_total",
			Help:      "Number of errors that occur during handling of jobs",
		}, []string{"type"})
	lastRequestMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "agent",
			Subsystem: "http",
			Name:      "last_request_time_seconds",
			Help:      "Unix/epoch time of the last HTTP request on /run.",
		},
	)
)
