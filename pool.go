package main

import (
	"context"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

func fillPool(ctx context.Context, WarmContainers chan<- runningContainer) {
	for {
		select {
		case <-ctx.Done():
			// WarmContainers will be cleaned up.
			return
		default:
			container, err := createContainer(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to create container")
				time.Sleep(time.Second)
				continue
			}

			log.WithField("containerID", container.containerID).Info("New container created and started")

			err = waitForContainerToBoot(ctx, container)
			if err != nil {
				log.WithError(err).Error("Container not available")
				_ = container.shutDown(ctx)
				continue
			}

			// Add the new container to the pool.
			// If the pool is full, this line will block until a slot is available.
			WarmContainers <- *container
		}
	}
}

func waitForContainerToBoot(ctx context.Context, container *runningContainer) error {
	// If the container is not available after 10s, move on.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// Query the endpoint until it provides a valid response.
	for {
		select {
		case <-ctx.Done():
			// Timeout reached.
			return ctx.Err()
		default:
			res, err := http.Get("http://" + container.addr + "/_/health")
			if err != nil {
				log.WithError(err).Error("Container agent not ready yet")
				time.Sleep(time.Second)
				continue
			}

			if res.StatusCode != 200 {
				log.WithField("containerID", container.containerID).Info("Container agent not ready yet")
			} else {
				log.WithField("containerID", container.containerID).Info("Container agent ready")
				return nil
			}

			time.Sleep(time.Second)
		}

	}
}
