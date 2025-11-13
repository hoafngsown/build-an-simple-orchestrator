package worker

import (
	"Mine-Cube/logger"
	"Mine-Cube/task"
	httputil "Mine-Cube/utils/http"
	"fmt"
	"net/http"
)

var handlerLog = logger.GetLogger("worker.api")

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	te, err := httputil.DecodeJSON[task.TaskEvent](r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}

	a.Worker.AddTask(te.Task)
	handlerLog.WithField("task_id", te.Task.ID).Info("Task added via API")
	httputil.WriteJSON(w, http.StatusCreated, te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, a.Worker.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	tID, err := httputil.GetUUIDParam(r, "taskID")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("No taskID passed in request: %v", err))
		return
	}

	taskToStop, ok := a.Worker.Db[tID]
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, fmt.Sprintf("No task found with ID: %v", tID))
		return
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	handlerLog.WithFields(map[string]interface{}{
		"task_id":      taskToStop.ID,
		"container_id": taskToStop.ContainerID,
	}).Info("Task stop requested via API")

	httputil.WriteNoContent(w)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, a.Worker.Stats)
}
