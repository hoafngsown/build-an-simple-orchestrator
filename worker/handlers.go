package worker

import (
	"Mine-Cube/task"
	httputil "Mine-Cube/utils/http"
	"fmt"
	"log"
	"net/http"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	te, err := httputil.DecodeJSON[task.TaskEvent](r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v\n", te.Task.ID)
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

	log.Printf(
		"Added task %v to stop container %v\n",
		taskToStop.ID,
		taskToStop.ContainerID,
	)

	httputil.WriteNoContent(w)
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, a.Worker.Stats)
}
