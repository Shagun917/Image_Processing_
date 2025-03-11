package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Store represents a store from the Store Master
type Store struct {
	StoreID   string `json:"store_id"`
	StoreName string `json:"store_name"`
	AreaCode  string `json:"area_code"`
}

// Visit represents a store visit with images
type Visit struct {
	StoreID   string   `json:"store_id"`
	ImageURLs []string `json:"image_url"`
	VisitTime string   `json:"visit_time"`
}

// SubmitJobRequest represents the request payload for job submission
type SubmitJobRequest struct {
	Count  int     `json:"count"`
	Visits []Visit `json:"visits"`
}

// JobResponse represents the response for job submission
type JobResponse struct {
	JobID int `json:"job_id"`
}

// JobStatusResponse represents the response for job status
type JobStatusResponse struct {
	Status string       `json:"status"`
	JobID  string       `json:"job_id"`
	Errors []StoreError `json:"error,omitempty"`
}

// StoreError represents an error for a specific store
type StoreError struct {
	StoreID string `json:"store_id"`
	Error   string `json:"error"`
}

// ImageResult represents the result of processing an image
type ImageResult struct {
	StoreID   string  `json:"store_id"`
	StoreName string  `json:"store_name"`
	AreaCode  string  `json:"area_code"`
	ImageURL  string  `json:"image_url"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Perimeter float64 `json:"perimeter"`
}

type JobData struct {
	ID          int
	Status      string
	Results     []ImageResult
	Errors      []StoreError
	CreatedAt   time.Time
	CompletedAt time.Time
	mu          sync.Mutex
}

var (
	jobs        = make(map[int]*JobData)
	jobsMutex   sync.Mutex
	nextJobID   = 1
	storeMaster = map[string]Store{
		"S00339218": {StoreID: "S00339218", StoreName: "Store A", AreaCode: "NYC"},
		"S01408764": {StoreID: "S01408764", StoreName: "Store B", AreaCode: "LA"},
	}
)

// getStore retrieves a store from the Store Master by ID
func getStore(storeID string) (Store, bool) {
	store, ok := storeMaster[storeID]
	return store, ok
}

func downloadAndGetDimensions(url string) (width, height int, err error) {

	// Create a temporary directory for downloads if it doesn't exist
	tempDir := "temp_images"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, 0755)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("error creating request: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("error downloading image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("error downloading image: status code %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("error decoding image: %v", err)
	}

	bounds := img.Bounds()
	width = bounds.Max.X - bounds.Min.X
	height = bounds.Max.Y - bounds.Min.Y

	return width, height, nil
}

func calculateImagePerimeter(storeID, imageURL string) (ImageResult, error) {

	store, exists := getStore(storeID)
	if !exists {
		return ImageResult{}, fmt.Errorf("store ID %s does not exist", storeID)
	}

	width, height, err := downloadAndGetDimensions(imageURL)
	if err != nil {
		return ImageResult{}, err
	}

	perimeter := 2.0 * float64(width+height)

	sleepTime := 100 + rand.Intn(300) // 0.1 to 0.4 seconds in milliseconds
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	return ImageResult{
		StoreID:   store.StoreID,
		StoreName: store.StoreName,
		AreaCode:  store.AreaCode,
		ImageURL:  imageURL,
		Width:     width,
		Height:    height,
		Perimeter: perimeter,
	}, nil
}

// processJob processes a job
func processJob(job *JobData, req SubmitJobRequest) {
	var wg sync.WaitGroup

	// Process each visit
	for _, visit := range req.Visits {
		storeID := visit.StoreID

		// Check if the store exists
		if _, exists := getStore(storeID); !exists {
			job.mu.Lock()
			job.Status = "failed"
			job.Errors = append(job.Errors, StoreError{
				StoreID: storeID,
				Error:   "Store ID does not exist",
			})
			job.mu.Unlock()
			job.CompletedAt = time.Now()
			return
		}

		// Process each image for this visit
		for _, imageURL := range visit.ImageURLs {
			wg.Add(1)
			go func(storeID, imageURL string) {
				defer wg.Done()

				result, err := calculateImagePerimeter(storeID, imageURL)
				job.mu.Lock()
				defer job.mu.Unlock()

				if err != nil {
					job.Status = "failed"
					job.Errors = append(job.Errors, StoreError{
						StoreID: storeID,
						Error:   err.Error(),
					})
					return
				}

				job.Results = append(job.Results, result)
			}(storeID, imageURL)
		}
	}

	// Wait for all image processing to complete
	wg.Wait()

	job.mu.Lock()
	if job.Status != "failed" {
		job.Status = "completed"
	}
	job.CompletedAt = time.Now()
	job.mu.Unlock()
}

func responseError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleSubmitJob handles the job submission endpoint
func handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		responseError(w, "Invalid Method")
		return
	}
	var req SubmitJobRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&req)
	if err != nil || (req.Count == 0 && len(req.Visits) > 0) {
		responseError(w, "Invalid request payload")
		return
	}

	if req.Count != len(req.Visits) {
		responseError(w, "Count does not match number of visits")
		return
	}

	// Create a new job
	jobsMutex.Lock()
	jobID := nextJobID
	nextJobID++
	job := &JobData{
		ID:        jobID,
		Status:    "ongoing",
		CreatedAt: time.Now(),
	}
	jobs[jobID] = job
	jobsMutex.Unlock()

	// Process the job asynchronously
	go processJob(job, req)

	// Return the job ID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(JobResponse{JobID: jobID})
}

// handleJobStatus handles the job status endpoint
func handleJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the job ID from the query parameters
	jobIDStr := r.URL.Query().Get("jobid")
	if jobIDStr == "" {
		http.Error(w, "Missing job ID", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Get the job
	jobsMutex.Lock()
	job, exists := jobs[jobID]
	jobsMutex.Unlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{}{})
		return
	}

	// Return the job status
	w.Header().Set("Content-Type", "application/json")
	response := JobStatusResponse{
		Status: job.Status,
		JobID:  strconv.Itoa(job.ID),
	}

	if job.Status == "failed" {
		response.Errors = job.Errors
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	// Initialize the random seed
	rand.Seed(time.Now().UnixNano())

	// Define the API routes
	http.HandleFunc("/submit/", handleSubmitJob)
	http.HandleFunc("/status", handleJobStatus)

	// Start the server
	port := 8080
	log.Printf("Server starting on port %d...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
