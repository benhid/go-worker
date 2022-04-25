package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type (
	Job struct {
		ID      string `json:"id"`
		Payload string `json:"payload"`
	}

	agentExecRes struct {
		Message      string `json:"message"`
		Error        string `json:"error"`
		StdErr       string `json:"stderr"`
		StdOut       string `json:"stdout"`
		ExecDuration int64  `json:"exec_duration"`
		MemUsage     int64  `json:"mem_usage"`
	}
)

func (job *Job) run(ctx context.Context, WarmContainers <-chan runningContainer) {
	log.WithField("ID", job.ID).Info("Starting job")

	// TODO - setjobReceived(ctx)

	// Get a ready-to-use container from the pool.
	container := <-WarmContainers
	defer container.shutDown(ctx)

	contextLogger := log.WithFields(
		log.Fields{
			"ID": job.ID, "containerID": container.containerID,
		},
	)
	contextLogger.Info("Handling job")

	// TODO - setjobRunning(ctx)

	var httpRes *http.Response
	var agentRes agentExecRes

	httpRes, err := http.Post("http://"+container.addr+"/", "application/json", bytes.NewBuffer([]byte(job.Payload)))
	if err != nil {
		log.WithError(err).Error("Failed to request execution to agent")
		return
	}

	if err = json.NewDecoder(httpRes.Body).Decode(&agentRes); err != nil {
		log.WithError(err).Error("Response decode failed")
		return
	}

	contextLogger.Info("Job execution finished")

	if httpRes.StatusCode != 200 {
		log.Error("Failed to run job")
		return
	}

	// TODO - setjobResult(ctx, agentRes)
}

// Validate job options.
func (job *Job) Validate() error {
	if job.ID == "" {
		return errors.New("ID must not be empty")
	}
	return nil
}
