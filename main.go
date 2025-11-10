package main

import (
	"Mine-Cube/task"
	"Mine-Cube/worker"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

// import (
// 	"Mine-Cube/manager"
// 	"Mine-Cube/node"
// 	"Mine-Cube/task"
// 	"Mine-Cube/worker"
// 	"fmt"
// 	"log"
// 	"os"
// 	"time"

// 	"github.com/docker/docker/client"
// 	"github.com/golang-collections/collections/queue"
// 	"github.com/google/uuid"
// )

// func main() {

// 	t := task.Task{
// 		ID:     uuid.New(),
// 		Name:   "Task-1",
// 		State:  task.Pending,
// 		Image:  "Image-1",
// 		Memory: 1024,
// 		Disk:   1,
// 	}

// 	te := task.TaskEvent{
// 		ID:        uuid.New(),
// 		State:     task.Pending,
// 		Timestamp: time.Now(),
// 		Task:      t,
// 	}

// 	fmt.Printf("task: %v\n", t)
// 	fmt.Printf("task event: %v\n", te)

// 	w := worker.Worker{
// 		Name:  "worker-1",
// 		Queue: *queue.New(),
// 		Db:    make(map[uuid.UUID]*task.Task),
// 	}
// 	fmt.Printf("worker: %v\n", w)
// 	w.CollectStats()
// 	w.RunTask()
// 	w.StartTask()
// 	w.StopTask(&t)

// 	m := manager.Manager{
// 		Pending: *queue.New(),
// 		TaskDb:  make(map[string][]*task.Task),
// 		EventDb: make(map[string][]*task.TaskEvent),
// 		Workers: []string{w.Name},
// 	}
// 	fmt.Printf("manager: %v\n", m)
// 	m.SelectWorker()
// 	m.UpdateTasks()
// 	m.SendWork()
// 	n := node.Node{
// 		Name:   "Node-1",
// 		Ip:     "192.168.1.1",
// 		Cores:  4,
// 		Memory: 1024,
// 		Disk:   25,
// 		Role:   "worker",
// 	}
// 	fmt.Printf("node: %v\n", n)

// 	fmt.Println("create a test container")

// 	dockerTask, createResult := createContainer()

// 	if createResult.Error != nil {
// 		log.Fatalf("Error creating container: %v", createResult.Error)
// 		os.Exit(1)
// 	}

// 	time.Sleep(time.Second * 10)

// 	fmt.Printf("stopping container %s\n", createResult.ContainerId)

// 	_, err := stopContainer(dockerTask, createResult.ContainerId)
// 	if err != nil {
// 		log.Fatalf("Error stopping container: %v", err)
// 		os.Exit(1)
// 	}
// }

// func createContainer() (*task.Docker, *task.DockerResult) {
// 	config := task.Config{
// 		Name:  "test-container-1",
// 		Image: "postgres:13",
// 		Env: []string{
// 			"POSTGRES_USER=cube",
// 			"POSTGRES_PASSWORD=secret",
// 		},
// 	}

// 	dc, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

// 	d := task.Docker{
// 		Client: dc,
// 		Config: config,
// 	}

// 	result := d.Run()

// 	if result.Error != nil {
// 		log.Fatalf("Error running container: %v", result.Error)
// 		return nil, nil
// 	}

// 	fmt.Printf("Container %s is running with config %v\n", result.ContainerId, config)
// 	return &d, &result
// }

// func stopContainer(d *task.Docker, id string) (*task.DockerResult, error) {
// 	result := d.Stop(id)
// 	if result.Error != nil {
// 		log.Fatalf("Error stopping container: %v", result.Error)
// 		return nil, result.Error
// 	}

// 	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerId)
// 	return &result, nil
// }

func main() {
	host := os.Getenv("HOST")
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	fmt.Println("Starting worker", host, ":", port)

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{
		Address: host,
		Port:    port,
		Worker:  &w,
	}

	go runTasks(&w)
	api.Start()
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() > 0 {
			result := w.RunTask()

			if result.Error != nil {
				log.Printf("Error running task: %v", result.Error)
			}
		} else {
			log.Println("No tasks to process currently")
		}

		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}
