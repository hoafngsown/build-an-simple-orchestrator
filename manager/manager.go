package manager

import (
	"Mine-Cube/task"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	// Pending: a queue of tasks that are waiting to be scheduled.
	Pending queue.Queue
	// TaskDb: a map of task names to tasks.
	TaskDb map[uuid.UUID]*task.Task
	// EventDb: a map of task names to task events.
	EventDb map[uuid.UUID]*task.TaskEvent
	// Workers: a list of worker names.
	Workers []string
	// WorkerTaskMap: a map of worker names to task IDs.
	WorkerTaskMap map[string][]uuid.UUID
	// TaskWorkerMap: a map of task IDs to worker names.
	TaskWorkerMap map[uuid.UUID]string
	// Index into worker slice
	LastWorker int
}

func NewManager(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		TaskDb:        taskDb,
		EventDb:       eventDb,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
	}
}

func (m *Manager) SelectWorker() string {
	// Currently we implement this function using Round Robin algorithm
	// TODO: Implement a more sophisticated algorithm
	var NextWorker int

	if m.LastWorker+1 < len(m.Workers) {
		NextWorker = m.LastWorker + 1
		m.LastWorker = NextWorker
	} else {
		NextWorker = 0
		m.LastWorker = NextWorker
	}

	return m.Workers[NextWorker]
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for task updates", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)

		if err != nil {
			log.Printf("Error connecting to %v: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sending request: %v\n", err)
		}

		d := json.NewDecoder(resp.Body)

		var tasks []*task.Task
		err = d.Decode(&tasks)

		if err != nil {
			log.Printf("Error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("Attempting to update task %v\n", t.ID)
			_, ok := m.TaskDb[t.ID]

			if !ok {
				log.Printf("Task with ID %s not found\n", t.ID)
				return
			}

			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() <= 0 {
		log.Println("No tasks to send")
		return
	}

	w := m.SelectWorker()

	e := m.Pending.Dequeue()
	te := e.(task.TaskEvent)
	t := te.Task
	log.Printf("Pulled %v off pending queue\n", t)

	m.EventDb[te.ID] = &te
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
	m.TaskWorkerMap[t.ID] = w

	t.State = task.Scheduled
	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Unable to marshal task object: %v.\n", t)
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.Printf("Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(te)
		return
	}

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		e := struct {
			HTTPStatusCode int    `json:"http_status_code"`
			Message        string `json:"message"`
		}{}

		err := d.Decode(&e)

		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
		return
	}

	t = task.Task{}
	err = d.Decode(&t)
	if err != nil {
		fmt.Printf("Error decoding response: %s\n", err.Error())
		return
	}

	log.Printf("%#v\n", t)
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}
	return tasks
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from workers")

		m.updateTasks()

		log.Println("Task updates completed")
		log.Println("Sleeping for 10 seconds")

		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")

		m.SendWork()

		log.Println("Sleeping for 10 seconds")

		time.Sleep(10 * time.Second)
	}
}
