package main

import (
	"context"
	"net"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

type ContainerCreateOpts struct {
	ContainerName    string
	ContainerConfig  *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
}

type runningContainer struct {
	containerCtx    context.Context
	containerCancel context.CancelFunc
	containerID     string
	addr            string
}

// agentImage to run workloads. The image has to expose a server listening on port 80 with at least two endpoints:
// * `/run` endpoint to handle job requests and
// * `/_/health` endpoint to check service status.
var agentImage string

// createContainerCfg returns container configuration with defaults.
func createContainerCfg() (ContainerCreateOpts, error) {
	// Allocate new free port on host.
	hostPort, err := GetFreePort()
	if err != nil {
		log.WithError(err).Error("Could not allocate new port")
		return ContainerCreateOpts{}, err
	}

	// Define a port opening for the image.
	containerPort, err := nat.NewPort("tcp", "80")
	if err != nil {
		log.WithError(err).Error("Unable to get the port")
		return ContainerCreateOpts{}, err
	}

	// Configure `hostConfig`.
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(hostPort),
				},
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	// Define ports to be exposed (has to be same as hostConfig.PortBindings.containerPort)
	exposedPorts := map[nat.Port]struct{}{
		containerPort: {},
	}

	// Configuration.
	config := &container.Config{
		Image:        agentImage + ":latest",
		ExposedPorts: exposedPorts,
	}

	return ContainerCreateOpts{
		ContainerConfig:  config,
		HostConfig:       hostConfig,
		NetworkingConfig: nil,
	}, nil
}

// createContainer starts a new container on host.
func createContainer(ctx context.Context) (*runningContainer, error) {
	containerCfg, err := createContainerCfg()
	if err != nil {
		log.WithError(err).Error("Failed creating container cfg")
		return nil, err
	}

	createResponse, err := client.ContainerCreate(
		ctx,
		containerCfg.ContainerConfig,
		containerCfg.HostConfig,
		containerCfg.NetworkingConfig,
		nil,
		"",
	)
	if err != nil {
		log.WithError(err).Error("Failed to setup container")
		return nil, err
	}

	log.WithField("containerID", createResponse.ID).Info("Starting container")

	if err := client.ContainerStart(ctx, createResponse.ID, types.ContainerStartOptions{}); err != nil {
		log.WithField("containerID", createResponse.ID).WithError(err).Error("Failed to start container")
		return nil, err
	}

	log.WithField("containerID", createResponse.ID).Info("Inspecting container information")

	resp, err := client.ContainerInspect(ctx, createResponse.ID)
	if err != nil {
		log.WithField("containerID", createResponse.ID).WithError(err).Error("Failed to inspect container")
		return nil, err
	}

	var ip, port string
	for _, bindings := range resp.NetworkSettings.Ports {
		for _, binding := range bindings {
			ip = binding.HostIP
			port = binding.HostPort
			break
		}
	}

	containerCtx, containerCancel := context.WithCancel(ctx)

	return &runningContainer{
		containerCtx:    containerCtx,
		containerCancel: containerCancel,
		containerID:     createResponse.ID,
		addr:            ip + ":" + port,
	}, nil
}

// shutDown stops and remove running container from host.
func (c runningContainer) shutDown(ctx context.Context) error {
	log.WithField("containerID", c.containerID).Info("Stopping container")

	removeOptions := types.ContainerRemoveOptions{Force: true}
	if err := client.ContainerRemove(ctx, c.containerID, removeOptions); err != nil {
		log.WithError(err).Error("Failed to remove container")
		return err
	}

	return nil
}

// GetFreePort returns a random free port.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
