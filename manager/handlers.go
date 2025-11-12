package manager

import (
	"Mine-Cube/task"
	httputil "Mine-Cube/utils/http"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	te, err := httputil.DecodeJSON[task.TaskEvent](r)
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
		return
	}

	a.Manager.AddTask(te)
	log.Printf("Added task %v\n", te.Task.ID)
	httputil.WriteJSON(w, http.StatusCreated, te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, a.Manager.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	tID, err := httputil.GetUUIDParam(r, "taskID")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, fmt.Sprintf("No taskID passed in request: %v", err))
		return
	}

	taskToStop, ok := a.Manager.TaskDb[tID]
	if !ok {
		httputil.WriteError(w, http.StatusNotFound, fmt.Sprintf("No task found with ID: %v", tID))
		return
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now(),
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy

	a.Manager.AddTask(te)

	log.Printf("Added task event %v to stop task %v\n", te.ID, taskToStop.ID)

	httputil.WriteNoContent(w)
}
