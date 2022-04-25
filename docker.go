package main

import (
	dockerClient "github.com/docker/docker/client"
)

var client *dockerClient.Client

func init() {
	var err error
	client, err = dockerClient.NewClientWithOpts(dockerClient.FromEnv)
	if err != nil {
		panic(err)
	}
}
