package worker

import (
	"Mine-Cube/task"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}

	err := d.Decode(&te)

	if err != nil {
		msg := fmt.Sprintf("error unmarshalling body: %v\n", err)
		log.Printf("%s\n", msg)

		w.WriteHeader(http.StatusBadRequest)
		e := struct {
			HTTPStatusCode int    `json:"http_status_code"`
			Message        string `json:"message"`
		}{
			HTTPStatusCode: 400,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())

}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	if taskID == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tID, _ := uuid.Parse(taskID)
	_, ok := a.Worker.Db[tID]

	if !ok {
		log.Printf("No task found with ID: %v\n", tID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	taskToStop := a.Worker.Db[tID]
	taskCopy := *taskToStop

	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	log.Printf(
		"Added task %v to stop container %v\n",
		taskToStop.ID,
		taskToStop.ContainerID,
	)

	w.WriteHeader(http.StatusNoContent)
}
