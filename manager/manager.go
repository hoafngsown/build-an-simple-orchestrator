package manager

import (
	"Mine-Cube/task"
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	// Pending: a queue of tasks that are waiting to be scheduled.
	Pending queue.Queue
	// TaskDb: a map of task names to tasks.
	TaskDb map[string][]*task.Task
	// EventDb: a map of task names to task events.
	EventDb map[string][]*task.TaskEvent
	// Workers: a list of worker names.
	Workers []string
	// WorkerTaskMap: a map of worker names to task IDs.
	WorkerTaskMap map[string][]uuid.UUID
	// TaskWorkerMap: a map of task IDs to worker names.
	TaskWorkerMap map[uuid.UUID]string
}

func (m *Manager) SelectWorker() {
	fmt.Println("I will select a worker")
}

func (m *Manager) UpdateTasks() {
	fmt.Println("I will update tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("I will send work to a worker")
}
