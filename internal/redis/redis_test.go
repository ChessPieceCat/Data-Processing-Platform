package redis

import (
	"context"
	"testing"
)

func TestOpenRedis(t *testing.T) {
	client := OpenRedis()
	if client == nil {
		t.Fatal("expected Redis client, got nil")
	}

	defer client.Close()

	if client.Options().Addr != "localhost:6379" {
		t.Fatalf(
			"expected Redis address %q, got %q",
			"localhost:6379",
			client.Options().Addr,
		)
	}
}

func TestPingRedis(t *testing.T) {
	client := OpenRedis()
	defer client.Close()

	if err := PingRedis(client); err != nil {
		t.Fatalf("expected Redis ping to succeed: %v", err)
	}
}

func TestInitializeGroup(t *testing.T) {
	client := OpenRedis()
	defer client.Close()

	if err := PingRedis(client); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	if err := InitializeGroup(client); err != nil {
		t.Fatalf("unexpected error initializing Redis group: %v", err)
	}

	groups, err := client.XInfoGroups(
		context.Background(),
		"job_queue",
	).Result()
	if err != nil {
		t.Fatalf("failed to inspect Redis consumer groups: %v", err)
	}

	found := false

	for _, group := range groups {
		if group.Name == "job_workers" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected job_workers consumer group to exist")
	}

	// Call again to verify that an already-existing group is handled.
	if err := InitializeGroup(client); err != nil {
		t.Fatalf(
			"expected InitializeGroup to succeed when group already exists: %v",
			err,
		)
	}
}
