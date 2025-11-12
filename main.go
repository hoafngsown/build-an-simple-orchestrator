package main

import (
	"Mine-Cube/manager"
	"Mine-Cube/task"
	"Mine-Cube/worker"
	"fmt"
	"os"
	"strconv"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	workerApi := setupWorker()
	setupManager(workerApi)
}

func setupWorker() *worker.Api {
	wh := os.Getenv("WORKER_HOST")
	wp, _ := strconv.Atoi(os.Getenv("WORKER_PORT"))

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.Api{Address: wh, Port: wp, Worker: &w}

	go w.RunTasks()
	go w.CollectStats()
	go w.UpdateTasks()
	go wapi.Start()

	return &wapi
}

func setupManager(workerApi *worker.Api) {
	mh := os.Getenv("MANAGER_HOST")
	mp, _ := strconv.Atoi(os.Getenv("MANAGER_PORT"))
	fmt.Println("manager: ", mh, ":", mp)

	workers := []string{fmt.Sprintf("%s:%d", workerApi.Address, workerApi.Port)}

	m := manager.NewManager(workers)
	mapi := manager.Api{Address: mh, Port: mp, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()
}
