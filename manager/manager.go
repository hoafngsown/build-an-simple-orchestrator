package manager

import (
	"Mine-Cube/logger"
	"Mine-Cube/task"
	httputil "Mine-Cube/utils/http"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

var log = logger.GetLogger("manager")

var PROCESS_TASKS_INTERVAL = 10 * time.Second
var UPDATE_TASKS_INTERVAL = 30 * time.Second
var HEALTH_CHECK_INTERVAL = 60 * time.Second

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

var MAX_RESTART_COUNT = 3

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
		log.WithField("worker", worker).Debug("Checking worker for task updates")

		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)

		if err != nil {
			log.WithField("worker", worker).Warnf("Error connecting to worker: %v", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.WithField("worker", worker).Warnf("Non-OK response from worker: %d", resp.StatusCode)
			continue
		}

		d := json.NewDecoder(resp.Body)

		var tasks []*task.Task
		err = d.Decode(&tasks)

		if err != nil {
			log.WithField("worker", worker).Errorf("Error unmarshalling tasks: %v", err)
			continue
		}

		for _, t := range tasks {
			log.WithField("task_id", t.ID).Debug("Updating task from worker")
			_, ok := m.TaskDb[t.ID]

			if !ok {
				log.WithField("task_id", t.ID).Error("Task not found in database")
				continue
			}

			if m.TaskDb[t.ID].State != t.State {
				log.WithFields(map[string]interface{}{
					"task_id":   t.ID,
					"old_state": m.TaskDb[t.ID].State,
					"new_state": t.State,
				}).Info("Task state changed")
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
			m.TaskDb[t.ID].HostPorts = t.HostPorts
		}
	}
}

func (m *Manager) SendWork() {
	if m.Pending.Len() <= 0 {
		log.Debug("No tasks in queue to send")
		return
	}

	w := m.SelectWorker()

	e := m.Pending.Dequeue()
	te := e.(task.TaskEvent)
	t := te.Task
	log.WithFields(map[string]interface{}{
		"task_id": t.ID,
		"worker":  w,
	}).Info("Scheduling task to worker")

	m.EventDb[te.ID] = &te
	m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
	m.TaskWorkerMap[t.ID] = w

	t.State = task.Scheduled
	m.TaskDb[t.ID] = &t

	data, err := json.Marshal(te)
	if err != nil {
		log.WithField("task_id", t.ID).Errorf("Failed to marshal task: %v", err)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))

	if err != nil {
		log.WithFields(map[string]interface{}{
			"worker":  w,
			"task_id": t.ID,
		}).Warnf("Failed to connect to worker, re-queueing task: %v", err)
		m.Pending.Enqueue(te)
		return
	}

	d := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		e := httputil.ErrorResponse{}

		err := d.Decode(&e)

		if err != nil {
			log.WithField("task_id", t.ID).Errorf("Failed to decode error response: %v", err)
			return
		}

		log.WithFields(map[string]interface{}{
			"status_code": e.HTTPStatusCode,
			"task_id":     t.ID,
		}).Errorf("Worker rejected task: %s", e.Message)
		return
	}

	t = task.Task{}
	err = d.Decode(&t)
	if err != nil {
		log.WithField("task_id", t.ID).Errorf("Failed to decode task response: %v", err)
		return
	}

	log.WithField("task_id", t.ID).Info("Task successfully sent to worker")
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
		log.WithFields(map[string]interface{}{
			"interval":     UPDATE_TASKS_INTERVAL,
			"worker_count": len(m.Workers),
			"task_count":   len(m.TaskDb),
		}).Debug("Checking for task updates from workers")

		m.updateTasks()

		time.Sleep(UPDATE_TASKS_INTERVAL)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.WithFields(map[string]interface{}{
			"interval":    PROCESS_TASKS_INTERVAL,
			"pending_len": m.Pending.Len(),
			"task_count":  len(m.TaskDb),
		}).Debug("Processing tasks in queue")

		m.SendWork()

		time.Sleep(PROCESS_TASKS_INTERVAL)
	}
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) checkTaskHealth(t task.Task) error {
	// Skip health check if no health check endpoint is configured
	if t.HealthCheck == "" {
		log.WithField("task_id", t.ID).Debug("No health check endpoint configured, skipping")
		return nil
	}

	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)

	// Skip health check if no host port is available
	if hostPort == nil {
		log.WithField("task_id", t.ID).Warn("No host port available for health check, skipping")
		return nil
	}

	worker := strings.Split(w, ":")

	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)

	log.WithFields(map[string]interface{}{
		"task_id": t.ID,
		"url":     url,
	}).Debug("Calling health check endpoint")

	resp, err := http.Get(url)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"task_id": t.ID,
			"url":     url,
		}).Warnf("Health check failed: %v", err)
		return fmt.Errorf("health check connection error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(map[string]interface{}{
			"task_id":     t.ID,
			"status_code": resp.StatusCode,
		}).Warn("Health check returned non-OK status")
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	log.WithField("task_id", t.ID).Debug("Health check passed")

	return nil
}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < MAX_RESTART_COUNT {
			err := m.checkTaskHealth(*t)
			if err != nil {
				if t.RestartCount < MAX_RESTART_COUNT {
					m.restartTask(t)
				}
			}
		} else if t.State == task.Failed && t.RestartCount < MAX_RESTART_COUNT {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.ID] = t

	log.WithFields(map[string]interface{}{
		"task_id":       t.ID,
		"restart_count": t.RestartCount,
		"worker":        w,
	}).Info("Restarting task")

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}

	data, err := json.Marshal(te)
	if err != nil {
		log.WithField("task_id", t.ID).Errorf("Failed to marshal task for restart: %v", err)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.WithFields(map[string]interface{}{
			"task_id": t.ID,
			"worker":  w,
		}).Warnf("Failed to restart task, re-queueing: %v", err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		e := httputil.ErrorResponse{}

		err := d.Decode(&e)

		if err != nil {
			log.WithField("task_id", t.ID).Errorf("Failed to decode restart error response: %v", err)
			return
		}

		log.WithFields(map[string]interface{}{
			"task_id":     t.ID,
			"status_code": e.HTTPStatusCode,
		}).Errorf("Worker rejected task restart: %s", e.Message)
		return
	}

	newTask := task.Task{}
	err = d.Decode(&newTask)

	if err != nil {
		log.WithField("task_id", t.ID).Errorf("Failed to decode restart response: %v", err)
		return
	}

	log.WithField("task_id", t.ID).Info("Task restarted successfully")
}

func (m *Manager) DoHealthChecks() {
	for {
		log.WithFields(map[string]interface{}{
			"interval":   HEALTH_CHECK_INTERVAL,
			"task_count": len(m.TaskDb),
		}).Debug("Performing task health checks")

		m.doHealthChecks()

		time.Sleep(HEALTH_CHECK_INTERVAL)
	}
}
