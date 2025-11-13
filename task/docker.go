package task

import (
	"Mine-Cube/logger"
	"context"
	"io"
	"math"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

var log = logger.GetLogger("docker")

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

type DockerInspectResponse struct {
	Error     error
	Container *container.InspectResponse
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
		log.Errorf("Failed to create Docker client: %v", err)
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

	log.WithField("image", d.Config.Image).Info("Pulling Docker image")

	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})

	if err != nil {
		log.WithField("image", d.Config.Image).Errorf("Failed to pull image: %v", err)
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
	log.WithField("name", d.Config.Name).Info("Creating container")

	res, err := d.Client.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, d.Config.Name)
	if err != nil {
		log.WithField("name", d.Config.Name).Errorf("Failed to create container: %v", err)
		return DockerResult{Error: err}
	}

	if len(res.Warnings) > 0 {
		log.WithFields(map[string]interface{}{
			"name":     d.Config.Name,
			"warnings": res.Warnings,
		}).Warn("Container created with warnings")
	}

	containerID := res.ID

	// Starting container
	log.WithField("container_id", containerID).Info("Starting container")

	err = d.Client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		log.WithField("container_id", containerID).Errorf("Failed to start container: %v", err)
		return DockerResult{Error: err}
	}

	// d.Config.Runtime.ContainerID = res.ID

	out, err := d.Client.ContainerLogs(
		ctx,
		containerID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		log.WithField("container_id", containerID).Errorf("Failed to get container logs: %v", err)
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

	log.WithField("container_id", id).Info("Stopping container")

	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		log.WithField("container_id", id).Errorf("Failed to stop container: %v", err)
		return DockerResult{Error: err}
	}

	log.WithField("container_id", id).Info("Removing container")

	err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		log.WithField("container_id", id).Errorf("Failed to remove container: %v", err)
		return DockerResult{Error: err}
	}

	log.WithField("container_id", id).Info("Container stopped and removed successfully")

	return DockerResult{
		ContainerId: id,
		Action:      "stop",
		Result:      "success",
		Error:       nil,
	}
}

func (d *Docker) Inspect(containerID string) DockerInspectResponse {
	dc, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	ctx := context.Background()

	resp, err := dc.ContainerInspect(ctx, containerID)

	if err != nil {
		log.WithField("container_id", containerID).Errorf("Failed to inspect container: %v", err)
		return DockerInspectResponse{Error: err}
	}

	return DockerInspectResponse{Container: &resp}
}
