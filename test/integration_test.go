package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

// TestResult tracks individual test results
type TestResult struct {
	Category     string
	TestName     string
	Method       string
	Endpoint     string
	StatusCode   int
	ExpectedCode int
	Passed       bool
	Duration     time.Duration
	Error        string
}

// Task represents the API task model
type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTaskRequest is the request body for creating tasks
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// UpdateTaskRequest is the request body for updating tasks
type UpdateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
}

var results []TestResult

func TestMain(m *testing.M) {
	// Check if server is reachable before running tests
	if err := checkServerHealth(); err != nil {
		fmt.Println("\n❌ ERROR: Cannot connect to API server")
		fmt.Printf("Please ensure the server is running on %s\n", baseURL)
		fmt.Println("\nStart the server with:")
		fmt.Println("  make run-dev")
		fmt.Println("  OR")
		fmt.Println("  docker-compose up")
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 120))
	fmt.Println("🚀 TASK MANAGEMENT API - INTEGRATION TEST SUITE")
	fmt.Println(strings.Repeat("=", 120))
	fmt.Printf("Base URL: %s\n", baseURL)
	fmt.Println(strings.Repeat("=", 120))
	fmt.Println("✅ Server is reachable")
	fmt.Println()

	// Run tests
	code := m.Run()

	// Print summary
	printSummary()

	os.Exit(code)
}

func checkServerHealth() error {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(baseURL + "/tasks")
	return err
}

func runTest(t *testing.T, category, testName, method, endpoint string, expectedCode int, testFunc func() (*http.Response, error)) bool {
	t.Helper()
	start := time.Now()

	resp, err := testFunc()
	duration := time.Since(start)

	result := TestResult{
		Category:     category,
		TestName:     testName,
		Method:       method,
		Endpoint:     endpoint,
		ExpectedCode: expectedCode,
		Duration:     duration,
	}

	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		results = append(results, result)
		t.Errorf("%s - %s: %v", category, testName, err)
		return false
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Passed = resp.StatusCode == expectedCode

	if !result.Passed {
		result.Error = fmt.Sprintf("Expected %d, got %d", expectedCode, resp.StatusCode)
		t.Errorf("%s - %s: expected status %d, got %d", category, testName, expectedCode, resp.StatusCode)
	}

	results = append(results, result)
	return result.Passed
}

func TestCreateTasks(t *testing.T) {
	// Test 1: Valid task with description
	var task1ID int64
	runTest(t, "CREATE", "Valid task with description", "POST", "/tasks", 201, func() (*http.Response, error) {
		payload := CreateTaskRequest{
			Title:       "Complete project documentation",
			Description: "Write comprehensive API documentation",
		}
		resp, err := makeRequest("POST", "/tasks", payload)
		if err == nil && resp.StatusCode == 201 {
			var task Task
			json.NewDecoder(resp.Body).Decode(&task)
			task1ID = task.ID
		}
		return resp, err
	})

	// Test 2: Valid task without description
	var task2ID int64
	runTest(t, "CREATE", "Valid task without description", "POST", "/tasks", 201, func() (*http.Response, error) {
		payload := CreateTaskRequest{
			Title: "Review pull requests",
		}
		resp, err := makeRequest("POST", "/tasks", payload)
		if err == nil && resp.StatusCode == 201 {
			var task Task
			json.NewDecoder(resp.Body).Decode(&task)
			task2ID = task.ID
		}
		return resp, err
	})

	// Test 3: Missing title
	runTest(t, "CREATE", "Missing title (validation)", "POST", "/tasks", 400, func() (*http.Response, error) {
		payload := map[string]string{"description": "No title"}
		return makeRequest("POST", "/tasks", payload)
	})

	// Test 4: Empty title
	runTest(t, "CREATE", "Empty title (validation)", "POST", "/tasks", 400, func() (*http.Response, error) {
		payload := CreateTaskRequest{
			Title:       "   ",
			Description: "Empty",
		}
		return makeRequest("POST", "/tasks", payload)
	})

	// Test 5: Invalid JSON
	runTest(t, "CREATE", "Malformed JSON", "POST", "/tasks", 400, func() (*http.Response, error) {
		req, _ := http.NewRequest("POST", baseURL+"/tasks", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		return http.DefaultClient.Do(req)
	})

	// Store IDs for subsequent tests
	t.Run("SubsequentTests", func(t *testing.T) {
		if task1ID > 0 && task2ID > 0 {
			testListTasks(t)
			testGetTask(t, task1ID)
			testUpdateTask(t, task1ID)
			testDeleteTask(t, task2ID)
		} else {
			t.Skip("Skipping remaining tests due to task creation failure")
		}
	})
}

func testListTasks(t *testing.T) {
	runTest(t, "LIST", "Get all tasks", "GET", "/tasks", 200, func() (*http.Response, error) {
		return http.Get(baseURL + "/tasks")
	})
}

func testGetTask(t *testing.T, taskID int64) {
	// Test 1: Get existing task
	runTest(t, "GET", "Get existing task", "GET", fmt.Sprintf("/tasks/%d", taskID), 200, func() (*http.Response, error) {
		return http.Get(fmt.Sprintf("%s/tasks/%d", baseURL, taskID))
	})

	// Test 2: Get non-existent task
	runTest(t, "GET", "Get non-existent task", "GET", "/tasks/9999", 404, func() (*http.Response, error) {
		return http.Get(baseURL + "/tasks/9999")
	})

	// Test 3: Invalid ID
	runTest(t, "GET", "Invalid task ID", "GET", "/tasks/abc", 400, func() (*http.Response, error) {
		return http.Get(baseURL + "/tasks/abc")
	})
}

func testUpdateTask(t *testing.T, taskID int64) {
	// Test 1: Valid update to done
	runTest(t, "UPDATE", "Update task to done", "PUT", fmt.Sprintf("/tasks/%d", taskID), 200, func() (*http.Response, error) {
		payload := UpdateTaskRequest{
			Title:       "Updated task",
			Description: "Updated desc",
			Status:      "done",
		}
		return makeRequest("PUT", fmt.Sprintf("/tasks/%d", taskID), payload)
	})

	// Test 2: Update back to todo
	runTest(t, "UPDATE", "Update task to todo", "PUT", fmt.Sprintf("/tasks/%d", taskID), 200, func() (*http.Response, error) {
		payload := UpdateTaskRequest{
			Title:       "Updated task",
			Description: "Back to todo",
			Status:      "todo",
		}
		return makeRequest("PUT", fmt.Sprintf("/tasks/%d", taskID), payload)
	})

	// Test 3: Invalid status
	runTest(t, "UPDATE", "Invalid status (validation)", "PUT", fmt.Sprintf("/tasks/%d", taskID), 400, func() (*http.Response, error) {
		payload := map[string]string{
			"title":  "Test",
			"status": "in-progress",
		}
		return makeRequest("PUT", fmt.Sprintf("/tasks/%d", taskID), payload)
	})

	// Test 4: Missing title
	runTest(t, "UPDATE", "Missing title (validation)", "PUT", fmt.Sprintf("/tasks/%d", taskID), 400, func() (*http.Response, error) {
		payload := map[string]string{
			"description": "No title",
			"status":      "done",
		}
		return makeRequest("PUT", fmt.Sprintf("/tasks/%d", taskID), payload)
	})

	// Test 5: Update non-existent task
	runTest(t, "UPDATE", "Update non-existent task", "PUT", "/tasks/9999", 404, func() (*http.Response, error) {
		payload := UpdateTaskRequest{
			Title:  "Test",
			Status: "done",
		}
		return makeRequest("PUT", "/tasks/9999", payload)
	})
}

func testDeleteTask(t *testing.T, taskID int64) {
	// Test 1: Delete existing task
	runTest(t, "DELETE", "Delete existing task", "DELETE", fmt.Sprintf("/tasks/%d", taskID), 204, func() (*http.Response, error) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/tasks/%d", baseURL, taskID), nil)
		return http.DefaultClient.Do(req)
	})

	// Test 2: Verify task is deleted
	runTest(t, "DELETE", "Verify task deleted", "GET", fmt.Sprintf("/tasks/%d", taskID), 404, func() (*http.Response, error) {
		return http.Get(fmt.Sprintf("%s/tasks/%d", baseURL, taskID))
	})

	// Test 3: Delete non-existent task
	runTest(t, "DELETE", "Delete non-existent task", "DELETE", "/tasks/9999", 404, func() (*http.Response, error) {
		req, _ := http.NewRequest("DELETE", baseURL+"/tasks/9999", nil)
		return http.DefaultClient.Do(req)
	})

	// Test 4: Invalid ID
	runTest(t, "DELETE", "Invalid task ID", "DELETE", "/tasks/abc", 400, func() (*http.Response, error) {
		req, _ := http.NewRequest("DELETE", baseURL+"/tasks/abc", nil)
		return http.DefaultClient.Do(req)
	})
}

func makeRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, body)
	if err != nil {
		return nil, err
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

func printSummary() {
	if len(results) == 0 {
		fmt.Println("No tests were run.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 120))
	fmt.Println("TEST RESULTS SUMMARY")
	fmt.Println(strings.Repeat("=", 120))

	// Print table header
	fmt.Printf("%-4s | %-10s | %-30s | %-6s | %-15s | %-10s | %-10s | %-10s\n",
		"#", "Category", "Test Name", "Method", "Endpoint", "Status", "Code", "Time(ms)")
	fmt.Println(strings.Repeat("-", 120))

	// Print results
	for i, result := range results {
		status := "✅ PASS"
		if !result.Passed {
			status = "❌ FAIL"
		}
		code := fmt.Sprintf("%d/%d", result.StatusCode, result.ExpectedCode)
		duration := fmt.Sprintf("%.2f", float64(result.Duration.Microseconds())/1000.0)

		fmt.Printf("%-4d | %-10s | %-30s | %-6s | %-15s | %-10s | %-10s | %-10s\n",
			i+1, result.Category, result.TestName, result.Method, result.Endpoint,
			status, code, duration)
	}

	// Calculate statistics
	total := len(results)
	passed := 0
	var totalDuration time.Duration
	for _, r := range results {
		if r.Passed {
			passed++
		}
		totalDuration += r.Duration
	}
	failed := total - passed

	fmt.Println("\n" + strings.Repeat("=", 120))
	fmt.Println("📊 STATISTICS")
	fmt.Println(strings.Repeat("=", 120))

	successRate := float64(passed) / float64(total) * 100
	avgDuration := float64(totalDuration.Microseconds()) / float64(total) / 1000.0

	fmt.Printf("  %-20s %d\n", "Total Tests", total)
	fmt.Printf("  %-20s ✅ %d\n", "Passed", passed)
	fmt.Printf("  %-20s ❌ %d\n", "Failed", failed)
	fmt.Printf("  %-20s %.1f%%\n", "Success Rate", successRate)
	fmt.Printf("  %-20s %.2fms\n", "Total Duration", float64(totalDuration.Microseconds())/1000.0)
	fmt.Printf("  %-20s %.2fms\n", "Average Duration", avgDuration)

	fmt.Println(strings.Repeat("=", 120))

	// Overall result
	if failed == 0 {
		fmt.Println("\n✅ ALL TESTS PASSED! 🎉")
	} else {
		fmt.Printf("\n❌ %d TEST(S) FAILED\n", failed)
		fmt.Println("\nFailed tests:")
		for i, result := range results {
			if !result.Passed {
				fmt.Printf("  %d. %s - %s\n", i+1, result.Category, result.TestName)
				if result.Error != "" {
					fmt.Printf("     Error: %s\n", result.Error)
				}
			}
		}
	}
	fmt.Println()
}
