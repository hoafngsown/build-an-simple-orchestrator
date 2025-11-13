package worker

import (
	"Mine-Cube/logger"
	"Mine-Cube/task"
	"errors"
	"fmt"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

var log = logger.GetLogger("worker")

var COLLECT_STATS_INTERVAL = 60 * time.Second
var RUN_TASKS_INTERVAL = 10 * time.Second
var UPDATE_TASKS_INTERVAL = 30 * time.Second

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *Stats
}

func (w *Worker) CollectStats() {
	for {
		log.WithField("interval", COLLECT_STATS_INTERVAL).Debug("Collecting system stats")

		w.Stats = GetStats()
		w.Stats.TaskCount = w.TaskCount

		time.Sleep(COLLECT_STATS_INTERVAL)
	}
}

func (w *Worker) updateTasks() {
	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				log.WithField("task_id", id).Errorf("Error inspecting container: %v", resp.Error)
			}

			if resp.Container == nil {
				log.WithField("task_id", id).Warn("No container found for running task, marking as failed")
				w.Db[id].State = task.Failed
			}

			if resp.Container != nil && resp.Container.State.Status == "exited" {
				log.WithFields(map[string]interface{}{
					"task_id": id,
					"status":  resp.Container.State.Status,
				}).Warn("Container exited, marking task as failed")
				w.Db[id].State = task.Failed
			}

			if resp.Container != nil {
				w.Db[id].HostPorts =
					resp.Container.NetworkSettings.NetworkSettingsBase.Ports
			}
		}
	}
}

func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()

	if t == nil {
		log.Debug("No tasks in queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskPersisted := w.Db[taskQueued.ID]

	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.ID] = taskPersisted
	}

	log.WithFields(map[string]interface{}{
		"task_id":         taskQueued.ID,
		"queued_state":    taskQueued.State,
		"persisted_state": taskPersisted.State,
	}).Debug("Processing task from queue")

	// 3 Retrieve the task from the worker's Db.
	var result task.DockerResult

	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(*taskPersisted)
		case task.Completed:
			result = w.StopTask(*taskPersisted)
		default:
			result.Error = errors.New("we should not get here")
		}
	} else {
		err := fmt.Errorf("invalid state transition for task %v: %v -> %v", taskPersisted.ID, taskPersisted.State, taskQueued.State)
		log.WithFields(map[string]interface{}{
			"task_id":    taskPersisted.ID,
			"from_state": taskPersisted.State,
			"to_state":   taskQueued.State,
		}).Error("Invalid state transition")
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()

	log.WithField("task_id", t.ID).Info("Starting task")

	taskConfig := task.NewConfig(&t)
	docker := task.NewDocker(taskConfig)

	if docker == nil {
		err := errors.New("failed to create Docker client")
		log.WithField("task_id", t.ID).Errorf("Failed to create Docker client: %v", err)
		return task.DockerResult{Error: err}
	}

	result := docker.Run()

	if result.Error != nil {
		log.WithField("task_id", t.ID).Errorf("Failed to run task: %v", result.Error)
		t.State = task.Failed
		w.Db[t.ID] = &t
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db[t.ID] = &t

	log.WithFields(map[string]interface{}{
		"task_id":      t.ID,
		"container_id": result.ContainerId,
	}).Info("Task started successfully")

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	log.WithFields(map[string]interface{}{
		"task_id":      t.ID,
		"container_id": t.ContainerID,
	}).Info("Stopping task")

	taskConfig := task.NewConfig(&t)
	docker := task.NewDocker(taskConfig)

	if docker == nil {
		err := errors.New("failed to create Docker client")
		log.WithField("task_id", t.ID).Errorf("Failed to create Docker client: %v", err)
		return task.DockerResult{Error: err}
	}

	result := docker.Stop(t.ContainerID)
	if result.Error != nil {
		log.WithField("container_id", t.ContainerID).Errorf("Error stopping container: %v", result.Error)
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db[t.ID] = &t

	log.WithFields(map[string]interface{}{
		"task_id":      t.ID,
		"container_id": t.ContainerID,
	}).Info("Task stopped and removed successfully")

	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []task.Task {
	tasks := make([]task.Task, 0, len(w.Db))

	for _, t := range w.Db {
		tasks = append(tasks, *t)
	}

	return tasks
}

func (w *Worker) RunTasks() {
	for {
		log.WithFields(map[string]interface{}{
			"interval":    RUN_TASKS_INTERVAL,
			"queue_len":   w.Queue.Len(),
			"task_count":  len(w.Db),
		}).Debug("Processing task queue")

		if w.Queue.Len() > 0 {
			result := w.runTask()

			if result.Error != nil {
				log.Errorf("Error running task: %v", result.Error)
			}
		} else {
			log.Debug("No tasks to process currently")
		}

		time.Sleep(RUN_TASKS_INTERVAL)
	}
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
		log.WithFields(map[string]interface{}{
			"interval":   UPDATE_TASKS_INTERVAL,
			"task_count": len(w.Db),
		}).Debug("Checking status of tasks")

		w.updateTasks()

		time.Sleep(UPDATE_TASKS_INTERVAL)
	}
}
