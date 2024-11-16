package jobs

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // Support for JPEG decoding
	_ "image/png"  // Support for PNG decoding
	"io"
	"net/http"
	"sync"
	"time"
	"math/rand"
)

// Job structure
type Job struct {
	Status  string        `json:"status"`
	Errors  []interface{} `json:"errors,omitempty"`
	Results []interface{} `json:"results,omitempty"`
}

// Store structure
type Store struct {
	StoreID   string `json:"StoreID"`
	StoreName string `json:"StoreName"`
	AreaCode  string `json:"AreaCode"`
}

// Simulate GPU processing delay
func simulateProcessing() {
	delay := time.Duration(rand.Intn(300)+100) * time.Millisecond
	time.Sleep(delay)
}

// Download image and calculate dimensions
func processImage(imageURL string) (map[string]interface{}, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: %s", resp.Status)
	}

	// Read the image
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	// Decode image dimensions
	img, _, err := image.DecodeConfig(buf)
	if err != nil {
		return nil, errors.New("unable to calculate image dimensions")
	}

	perimeter := 2 * (img.Width + img.Height)
	simulateProcessing()

	return map[string]interface{}{
		"success":   true,
		"width":     img.Width,
		"height":    img.Height,
		"perimeter": perimeter,
	}, nil
}

// Process a job
func ProcessJob(visits []map[string]interface{}, storeMaster []Store, jobID string, jobs map[string]*Job, jobsMutex *sync.Mutex) {
	for _, visit := range visits {
		storeID, ok := visit["store_id"].(string)
		if !ok {
			setJobFailed(jobs, jobsMutex, jobID, storeID, "Invalid store_id format")
			return
		}

		// Validate store_id
		var store *Store
		for _, s := range storeMaster {
			if s.StoreID == storeID {
				store = &s
				break
			}
		}
		if store == nil {
			setJobFailed(jobs, jobsMutex, jobID, storeID, "Invalid store_id")
			return
		}

		// Process images
		imageURLs, ok := visit["image_url"].([]interface{})
		if !ok {
			setJobFailed(jobs, jobsMutex, jobID, storeID, "Invalid image_url format")
			return
		}

		for _, url := range imageURLs {
			imageURL, ok := url.(string)
			if !ok {
				setJobFailed(jobs, jobsMutex, jobID, storeID, "Invalid image_url format")
				return
			}

			result, err := processImage(imageURL)
			if err != nil {
				setJobFailed(jobs, jobsMutex, jobID, storeID, err.Error())
				return
			}

			jobsMutex.Lock()
			jobs[jobID].Results = append(jobs[jobID].Results, map[string]interface{}{
				"store_id":   storeID,
				"store_name": store.StoreName,
				"area_code":  store.AreaCode,
				"visit_time": visit["visit_time"],
				"image_url":  imageURL,
				"perimeter":  result["perimeter"],
			})
			jobsMutex.Unlock()
		}
	}

	jobsMutex.Lock()
	jobs[jobID].Status = "completed"
	jobsMutex.Unlock()
}

// Helper to mark job as failed
func setJobFailed(jobs map[string]*Job, jobsMutex *sync.Mutex, jobID, storeID, errMsg string) {
	jobsMutex.Lock()
	defer jobsMutex.Unlock()
	jobs[jobID].Status = "failed"
	jobs[jobID].Errors = append(jobs[jobID].Errors, map[string]string{
		"store_id": storeID,
		"error":    errMsg,
	})
}
