package task

import (
	"context"
	"io"
	"log"
	"math"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

type Config struct {
	// Name of the container
	Name string
	// Attach to the container's stdin
	AttachStdin bool
	// Attach to the container's stdout
	AttachStdout bool
	// Attach to the container's stderr
	AttachStderr bool
	// Exposed ports to the container
	ExposedPorts nat.PortSet
	// Command to run in the container
	Cmd []string
	// Image to use for the container
	Image string
	// CPU to use for the container
	Cpu float64
	// Memory to use for the container
	Memory int64
	// Disk to use for the container
	Disk int64
	// Environment variables to set in the container
	Env []string
	// Restart policy to use for the container
	RestartPolicy string
}

type Docker struct {
	Client *client.Client
	Config Config
}

type DockerResult struct {
	Error error
	// could be start | stop
	Action      string
	ContainerId string
	// arbitrary text to provide information about the result
	Result string
}

func NewConfig(t *Task) Config {
	return Config{
		Name:          t.Name,
		ExposedPorts:  t.ExposedPorts,
		Image:         t.Image,
		RestartPolicy: t.RestartPolicy,
	}
}

func NewDocker(config Config) *Docker {
	dc, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil {
		log.Printf("Error creating Docker client: %v\n", err)
		return nil
	}

	return &Docker{
		Client: dc,
		Config: config,
	}
}

func (d *Docker) Run() DockerResult {
	// Pulling image from registry
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})

	if err != nil {
		log.Printf("Error pulling image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	// Print the progress of the image pull to the console
	io.Copy(os.Stdout, reader)

	// Configuring container
	restartPolicy := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	containerConfig := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}

	resources := container.Resources{
		Memory:   int64(d.Config.Memory),
		NanoCPUs: int64(d.Config.Cpu * math.Pow(10, 9)),
	}

	hostConfig := container.HostConfig{
		Resources:       resources,
		RestartPolicy:   restartPolicy,
		PublishAllPorts: true,
	}

	// Creating container
	res, err := d.Client.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf("Error creating container %s: %v\n", d.Config.Name, err)
		return DockerResult{Error: err}
	}

	log.Printf("Container created with warnings: %s\n", res.Warnings)

	containerID := res.ID

	// Starting container
	err = d.Client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", containerID, err)
		return DockerResult{Error: err}
	}

	// d.Config.Runtime.ContainerID = res.ID

	out, err := d.Client.ContainerLogs(
		ctx,
		containerID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", containerID, err)
		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return DockerResult{
		ContainerId: containerID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	ctx := context.Background()

	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{
		ContainerId: id,
		Action:      "stop",
		Result:      "success",
		Error:       nil,
	}
}
