#!/usr/bin/env python3
"""
Manual API testing script for the Task Management API.
Tests all endpoints with various scenarios.
"""

import requests
import json
from typing import Dict, Any

BASE_URL = "http://localhost:8080"

def print_response(method: str, endpoint: str, response: requests.Response, payload: Dict[str, Any] = None):
    """Pretty print the API response."""
    print(f"\n{'='*70}")
    print(f"{method} {endpoint}")
    if payload:
        print(f"Payload: {json.dumps(payload, indent=2)}")
    print(f"Status Code: {response.status_code}")
    if response.text:
        try:
            print(f"Response: {json.dumps(response.json(), indent=2)}")
        except:
            print(f"Response: {response.text}")
    print('='*70)

def test_create_task():
    """Test POST /tasks endpoint."""
    print("\n🧪 Testing: Create Task (POST /tasks)")

    # Valid task with description
    payload = {
        "title": "Complete project documentation",
        "description": "Write comprehensive API documentation"
    }
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    print_response("POST", "/tasks", response, payload)
    assert response.status_code == 201, f"Expected 201, got {response.status_code}"
    task1 = response.json()
    assert task1["id"] > 0
    assert task1["title"] == payload["title"]
    assert task1["status"] == "todo"
    print("✅ Valid task with description - PASSED")

    # Valid task without description
    payload = {"title": "Review pull requests"}
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    print_response("POST", "/tasks", response, payload)
    assert response.status_code == 201
    task2 = response.json()
    assert task2["description"] == ""
    print("✅ Valid task without description - PASSED")

    # Invalid - missing title
    payload = {"description": "No title provided"}
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    print_response("POST", "/tasks", response, payload)
    assert response.status_code == 400
    print("✅ Missing title validation - PASSED")

    # Invalid - empty title
    payload = {"title": "   ", "description": "Empty title"}
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    print_response("POST", "/tasks", response, payload)
    assert response.status_code == 400
    print("✅ Empty title validation - PASSED")

    return task1["id"], task2["id"]

def test_list_tasks():
    """Test GET /tasks endpoint."""
    print("\n🧪 Testing: List Tasks (GET /tasks)")

    response = requests.get(f"{BASE_URL}/tasks")
    print_response("GET", "/tasks", response)
    assert response.status_code == 200
    tasks = response.json()
    assert isinstance(tasks, list)
    assert len(tasks) >= 2
    print(f"✅ List tasks - PASSED (Found {len(tasks)} tasks)")

def test_get_task(task_id: int):
    """Test GET /tasks/{id} endpoint."""
    print("\n🧪 Testing: Get Task (GET /tasks/{id})")

    # Valid task ID
    response = requests.get(f"{BASE_URL}/tasks/{task_id}")
    print_response("GET", f"/tasks/{task_id}", response)
    assert response.status_code == 200
    task = response.json()
    assert task["id"] == task_id
    print("✅ Get existing task - PASSED")

    # Non-existent task
    response = requests.get(f"{BASE_URL}/tasks/9999")
    print_response("GET", "/tasks/9999", response)
    assert response.status_code == 404
    print("✅ Get non-existent task - PASSED")

    # Invalid ID
    response = requests.get(f"{BASE_URL}/tasks/abc")
    print_response("GET", "/tasks/abc", response)
    assert response.status_code == 400
    print("✅ Invalid task ID - PASSED")

def test_update_task(task_id: int):
    """Test PUT /tasks/{id} endpoint."""
    print("\n🧪 Testing: Update Task (PUT /tasks/{id})")

    # Valid update
    payload = {
        "title": "Updated task title",
        "description": "Updated description",
        "status": "done"
    }
    response = requests.put(f"{BASE_URL}/tasks/{task_id}", json=payload)
    print_response("PUT", f"/tasks/{task_id}", response, payload)
    assert response.status_code == 200
    updated = response.json()
    assert updated["title"] == payload["title"]
    assert updated["status"] == "done"
    print("✅ Valid update - PASSED")

    # Update back to todo
    payload = {
        "title": "Updated task title",
        "description": "Back to todo",
        "status": "todo"
    }
    response = requests.put(f"{BASE_URL}/tasks/{task_id}", json=payload)
    print_response("PUT", f"/tasks/{task_id}", response, payload)
    assert response.status_code == 200
    assert response.json()["status"] == "todo"
    print("✅ Update status to todo - PASSED")

    # Invalid status
    payload = {
        "title": "Test",
        "description": "Invalid status",
        "status": "in-progress"
    }
    response = requests.put(f"{BASE_URL}/tasks/{task_id}", json=payload)
    print_response("PUT", f"/tasks/{task_id}", response, payload)
    assert response.status_code == 400
    print("✅ Invalid status validation - PASSED")

    # Missing title
    payload = {
        "description": "No title",
        "status": "done"
    }
    response = requests.put(f"{BASE_URL}/tasks/{task_id}", json=payload)
    print_response("PUT", f"/tasks/{task_id}", response, payload)
    assert response.status_code == 400
    print("✅ Missing title validation - PASSED")

    # Non-existent task
    payload = {
        "title": "Test",
        "status": "done"
    }
    response = requests.put(f"{BASE_URL}/tasks/9999", json=payload)
    print_response("PUT", "/tasks/9999", response, payload)
    assert response.status_code == 404
    print("✅ Update non-existent task - PASSED")

def test_delete_task(task_id: int):
    """Test DELETE /tasks/{id} endpoint."""
    print("\n🧪 Testing: Delete Task (DELETE /tasks/{id})")

    # Valid delete
    response = requests.delete(f"{BASE_URL}/tasks/{task_id}")
    print_response("DELETE", f"/tasks/{task_id}", response)
    assert response.status_code == 204
    print("✅ Delete existing task - PASSED")

    # Verify task is deleted
    response = requests.get(f"{BASE_URL}/tasks/{task_id}")
    print_response("GET", f"/tasks/{task_id} (verify deletion)", response)
    assert response.status_code == 404
    print("✅ Verify task deleted - PASSED")

    # Delete non-existent task
    response = requests.delete(f"{BASE_URL}/tasks/9999")
    print_response("DELETE", "/tasks/9999", response)
    assert response.status_code == 404
    print("✅ Delete non-existent task - PASSED")

    # Invalid ID
    response = requests.delete(f"{BASE_URL}/tasks/abc")
    print_response("DELETE", "/tasks/abc", response)
    assert response.status_code == 400
    print("✅ Invalid task ID - PASSED")

def main():
    """Run all API tests."""
    print("🚀 Starting API Tests")
    print(f"Base URL: {BASE_URL}")

    try:
        # Test server is running
        requests.get(f"{BASE_URL}/tasks", timeout=2)
    except requests.exceptions.ConnectionError:
        print("\n❌ ERROR: Cannot connect to API server")
        print(f"Please ensure the server is running on {BASE_URL}")
        return

    try:
        # Run tests
        task1_id, task2_id = test_create_task()
        test_list_tasks()
        test_get_task(task1_id)
        test_update_task(task1_id)
        test_delete_task(task2_id)

        print("\n" + "="*70)
        print("✅ All tests PASSED!")
        print("="*70)

    except AssertionError as e:
        print(f"\n❌ Test FAILED: {e}")
    except Exception as e:
        print(f"\n❌ Unexpected error: {e}")

if __name__ == "__main__":
    main()
