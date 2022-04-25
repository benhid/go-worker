package main

import (
	"log"
	"testing"
)

func TestJob_Valid(t *testing.T) {
	job := &Job{
		ID: "1",
	}
	err := job.Validate()
	if err != nil {
		log.Fatalf("Job is invalid")
	}
}
