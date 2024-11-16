package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	// Import the jobs package
	"kstore/jobs" // Ensure the module name is correct
)

var (
	storeMaster []jobs.Store          
	jobMap      = make(map[string]*jobs.Job) 
	jobsMutex   = &sync.Mutex{}
)

func loadStoreMaster() error {
	data, err := ioutil.ReadFile("storeMaster.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &storeMaster)
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Count  int              `json:"count"`
		Visits []map[string]interface{} `json:"visits"` 
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil || request.Count != len(request.Visits) {
		http.Error(w, `{"error": "Invalid input data"}`, http.StatusBadRequest)
		return
	}

	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	job := &jobs.Job{Status: "ongoing", Errors: []interface{}{}, Results: []interface{}{}}



	jobsMutex.Lock()
	jobMap[jobID] = job
	jobsMutex.Unlock()

	// Call the ProcessJob function from the jobs package
	go jobs.ProcessJob(request.Visits, storeMaster, jobID, jobMap, jobsMutex)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("jobid")
	if jobID == "" {
		http.Error(w, `{}`, http.StatusBadRequest)
		return
	}

	jobsMutex.Lock()
	job, exists := jobMap[jobID]
	jobsMutex.Unlock()

	if !exists {
		http.Error(w, `{}`, http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"status":  job.Status,
		"job_id":  jobID,
	}
	if job.Status == "failed" {
		response["error"] = job.Errors
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	if err := loadStoreMaster(); err != nil {
		log.Fatalf("Failed to load store master data: %v", err)
	}

	http.HandleFunc("/api/submit/", submitHandler)
	http.HandleFunc("/api/status", statusHandler)

	port := "3000"
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
