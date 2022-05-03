package main

import (
	"context"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
)

func main() {
	if err := redisExample(); err != nil {
		log.Fatal(err)
	}

}

func redisExample() error {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return err
	}
	defer client.Close()

	// Create a new context with "testcontainerd" namespace
	ctx := namespaces.WithNamespace(context.Background(), "testcontainerd")

	// Pull image
	log.Printf("here")

	image, err := client.Pull(ctx, "docker.io/library/redis:alpine", containerd.WithPullUnpack)
	if err != nil {
		return err
	}

	log.Printf("Successfully pulled %s image ", image.Name())

	size, err := image.Size(ctx)
	if err != nil {
		return err
	}
	log.Printf("Successfully got image size %d", size)

	// Create an OCI spec for the image
	// Using the defaults here but there are Opts to modify the defaults

	// Create container
	container, err := client.NewContainer(ctx,
		"redis-server",
		containerd.WithImage(image),
		containerd.WithNewSnapshot("redis-server-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)))
	if err != nil {
		return err
	}

	defer container.Delete(ctx, containerd.WithSnapshotCleanup)
	log.Printf("Successfully created container with ID as %s and snapshot with ID redis-server-snapshot", container.ID())

	// Create a task
	// This will create a new container and put it in a CREATED state
	// One can use this state to setup net interfaces and monitors
	// and containerd uses this state to setup cgroup metrics and container exit-status watchers
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return err
	}

	defer task.Delete(ctx)

	// Now we wait for the task to exit
	exitStatus, err := task.Wait(ctx)
	if err != nil {
		return err
	}
	//	We wait before starting the container to prevent race conditions
	// In case the container exists before the listener can be setup
	if err := task.Start(ctx); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	status := <-exitStatus
	code, exitAt, err := status.Result()
	if err != nil {
		return err

	}
	fmt.Printf("Container task exited with status code %d, exited at %s", code, exitAt)

	return nil
}
