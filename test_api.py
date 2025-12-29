#!/usr/bin/env python3
"""
API Testing Script for Task Management API
Tests all endpoints and displays results in tabular format.

Usage:
    python3 test_api.py              # Run tests with summary table
    python3 test_api.py -v           # Run tests with verbose output
    python3 test_api.py --verbose    # Same as -v
"""

import requests
import json
import sys
from typing import Dict, Any, List, Tuple
from dataclasses import dataclass
from enum import Enum

# Try to import tabulate, fallback to simple table if not available
try:
    from tabulate import tabulate
    HAS_TABULATE = True
except ImportError:
    HAS_TABULATE = False
    print("⚠️  Install 'tabulate' for better formatting: pip install tabulate")
    print()

BASE_URL = "http://localhost:8080"


class TestStatus(Enum):
    """Test result status."""
    PASS = "✅ PASS"
    FAIL = "❌ FAIL"
    SKIP = "⏭️  SKIP"


@dataclass
class TestResult:
    """Represents a single test result."""
    category: str
    test_name: str
    method: str
    endpoint: str
    status_code: int
    expected_code: int
    status: TestStatus
    duration_ms: float = 0.0
    error: str = ""

    @property
    def passed(self) -> bool:
        return self.status == TestStatus.PASS


class TestRunner:
    """Manages test execution and result tracking."""

    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self.results: List[TestResult] = []

    def add_result(self, result: TestResult):
        """Add a test result."""
        self.results.append(result)
        if self.verbose:
            self._print_verbose(result)

    def _print_verbose(self, result: TestResult):
        """Print detailed test information."""
        print(f"\n{'='*70}")
        print(f"{result.status.value} {result.category} - {result.test_name}")
        print(f"  Method:   {result.method} {result.endpoint}")
        print(f"  Expected: {result.expected_code}, Got: {result.status_code}")
        if result.error:
            print(f"  Error:    {result.error}")
        print(f"  Duration: {result.duration_ms:.2f}ms")
        print('='*70)

    def run_test(self, category: str, test_name: str, method: str,
                 endpoint: str, expected_code: int, test_func) -> bool:
        """Run a single test and record result."""
        import time
        start = time.time()

        try:
            response = test_func()
            duration_ms = (time.time() - start) * 1000

            if response.status_code == expected_code:
                status = TestStatus.PASS
                error = ""
            else:
                status = TestStatus.FAIL
                error = f"Expected {expected_code}, got {response.status_code}"

            result = TestResult(
                category=category,
                test_name=test_name,
                method=method,
                endpoint=endpoint,
                status_code=response.status_code,
                expected_code=expected_code,
                status=status,
                duration_ms=duration_ms,
                error=error
            )
            self.add_result(result)
            return result.passed

        except Exception as e:
            duration_ms = (time.time() - start) * 1000
            result = TestResult(
                category=category,
                test_name=test_name,
                method=method,
                endpoint=endpoint,
                status_code=0,
                expected_code=expected_code,
                status=TestStatus.FAIL,
                duration_ms=duration_ms,
                error=str(e)
            )
            self.add_result(result)
            return False

    def print_summary(self):
        """Print test results in tabular format."""
        if not self.results:
            print("No tests were run.")
            return

        # Prepare table data
        headers = ["#", "Category", "Test Name", "Method", "Endpoint", "Status", "Code", "Time(ms)"]
        rows = []

        for idx, result in enumerate(self.results, 1):
            rows.append([
                idx,
                result.category,
                result.test_name,
                result.method,
                result.endpoint,
                result.status.value,
                f"{result.status_code}/{result.expected_code}",
                f"{result.duration_ms:.2f}"
            ])

        # Print table
        print("\n" + "="*120)
        print("TEST RESULTS SUMMARY")
        print("="*120)

        if HAS_TABULATE:
            print(tabulate(rows, headers=headers, tablefmt="grid"))
        else:
            # Fallback simple table
            self._print_simple_table(headers, rows)

        # Print statistics
        total = len(self.results)
        passed = sum(1 for r in self.results if r.passed)
        failed = total - passed
        total_time = sum(r.duration_ms for r in self.results)

        print("\n" + "="*120)
        print(f"📊 STATISTICS")
        print("="*120)

        stats = [
            ["Total Tests", total],
            ["Passed", f"✅ {passed}"],
            ["Failed", f"❌ {failed}"],
            ["Success Rate", f"{(passed/total)*100:.1f}%"],
            ["Total Duration", f"{total_time:.2f}ms"],
            ["Average Duration", f"{total_time/total:.2f}ms"]
        ]

        if HAS_TABULATE:
            print(tabulate(stats, tablefmt="simple"))
        else:
            for stat in stats:
                print(f"  {stat[0]:<20} {stat[1]}")

        print("="*120 + "\n")

        # Overall result
        if failed == 0:
            print("✅ ALL TESTS PASSED! 🎉")
        else:
            print(f"❌ {failed} TEST(S) FAILED")
            print("\nFailed tests:")
            for idx, result in enumerate(self.results, 1):
                if not result.passed:
                    print(f"  {idx}. {result.category} - {result.test_name}")
                    if result.error:
                        print(f"     Error: {result.error}")

    def _print_simple_table(self, headers: List[str], rows: List[List]):
        """Fallback simple table formatting."""
        # Calculate column widths
        col_widths = [len(h) for h in headers]
        for row in rows:
            for i, cell in enumerate(row):
                col_widths[i] = max(col_widths[i], len(str(cell)))

        # Print header
        header_line = " | ".join(h.ljust(col_widths[i]) for i, h in enumerate(headers))
        print(header_line)
        print("-" * len(header_line))

        # Print rows
        for row in rows:
            row_line = " | ".join(str(cell).ljust(col_widths[i]) for i, cell in enumerate(row))
            print(row_line)


def test_create_tasks(runner: TestRunner) -> Tuple[int, int]:
    """Test POST /tasks endpoint."""
    task1_id = task2_id = None

    # Test 1: Valid task with description
    def test1():
        payload = {
            "title": "Complete project documentation",
            "description": "Write comprehensive API documentation"
        }
        response = requests.post(f"{BASE_URL}/tasks", json=payload)
        if response.status_code == 201:
            nonlocal task1_id
            task1_id = response.json()["id"]
        return response

    runner.run_test("CREATE", "Valid task with description", "POST", "/tasks", 201, test1)

    # Test 2: Valid task without description
    def test2():
        payload = {"title": "Review pull requests"}
        response = requests.post(f"{BASE_URL}/tasks", json=payload)
        if response.status_code == 201:
            nonlocal task2_id
            task2_id = response.json()["id"]
        return response

    runner.run_test("CREATE", "Valid task without description", "POST", "/tasks", 201, test2)

    # Test 3: Missing title
    runner.run_test(
        "CREATE", "Missing title (validation)", "POST", "/tasks", 400,
        lambda: requests.post(f"{BASE_URL}/tasks", json={"description": "No title"})
    )

    # Test 4: Empty title
    runner.run_test(
        "CREATE", "Empty title (validation)", "POST", "/tasks", 400,
        lambda: requests.post(f"{BASE_URL}/tasks", json={"title": "   ", "description": "Empty"})
    )

    # Test 5: Invalid JSON
    runner.run_test(
        "CREATE", "Malformed JSON", "POST", "/tasks", 400,
        lambda: requests.post(f"{BASE_URL}/tasks", data="invalid json",
                            headers={"Content-Type": "application/json"})
    )

    return task1_id, task2_id


def test_list_tasks(runner: TestRunner):
    """Test GET /tasks endpoint."""
    runner.run_test(
        "LIST", "Get all tasks", "GET", "/tasks", 200,
        lambda: requests.get(f"{BASE_URL}/tasks")
    )


def test_get_task(runner: TestRunner, task_id: int):
    """Test GET /tasks/{id} endpoint."""
    # Test 1: Get existing task
    runner.run_test(
        "GET", "Get existing task", "GET", f"/tasks/{task_id}", 200,
        lambda: requests.get(f"{BASE_URL}/tasks/{task_id}")
    )

    # Test 2: Get non-existent task
    runner.run_test(
        "GET", "Get non-existent task", "GET", "/tasks/9999", 404,
        lambda: requests.get(f"{BASE_URL}/tasks/9999")
    )

    # Test 3: Invalid ID
    runner.run_test(
        "GET", "Invalid task ID", "GET", "/tasks/abc", 400,
        lambda: requests.get(f"{BASE_URL}/tasks/abc")
    )


def test_update_task(runner: TestRunner, task_id: int):
    """Test PUT /tasks/{id} endpoint."""
    # Test 1: Valid update to done
    runner.run_test(
        "UPDATE", "Update task to done", "PUT", f"/tasks/{task_id}", 200,
        lambda: requests.put(f"{BASE_URL}/tasks/{task_id}", json={
            "title": "Updated task",
            "description": "Updated desc",
            "status": "done"
        })
    )

    # Test 2: Update back to todo
    runner.run_test(
        "UPDATE", "Update task to todo", "PUT", f"/tasks/{task_id}", 200,
        lambda: requests.put(f"{BASE_URL}/tasks/{task_id}", json={
            "title": "Updated task",
            "description": "Back to todo",
            "status": "todo"
        })
    )

    # Test 3: Invalid status
    runner.run_test(
        "UPDATE", "Invalid status (validation)", "PUT", f"/tasks/{task_id}", 400,
        lambda: requests.put(f"{BASE_URL}/tasks/{task_id}", json={
            "title": "Test",
            "status": "in-progress"
        })
    )

    # Test 4: Missing title
    runner.run_test(
        "UPDATE", "Missing title (validation)", "PUT", f"/tasks/{task_id}", 400,
        lambda: requests.put(f"{BASE_URL}/tasks/{task_id}", json={
            "description": "No title",
            "status": "done"
        })
    )

    # Test 5: Update non-existent task
    runner.run_test(
        "UPDATE", "Update non-existent task", "PUT", "/tasks/9999", 404,
        lambda: requests.put(f"{BASE_URL}/tasks/9999", json={
            "title": "Test",
            "status": "done"
        })
    )


def test_delete_task(runner: TestRunner, task_id: int):
    """Test DELETE /tasks/{id} endpoint."""
    # Test 1: Delete existing task
    runner.run_test(
        "DELETE", "Delete existing task", "DELETE", f"/tasks/{task_id}", 204,
        lambda: requests.delete(f"{BASE_URL}/tasks/{task_id}")
    )

    # Test 2: Verify task is deleted
    runner.run_test(
        "DELETE", "Verify task deleted", "GET", f"/tasks/{task_id}", 404,
        lambda: requests.get(f"{BASE_URL}/tasks/{task_id}")
    )

    # Test 3: Delete non-existent task
    runner.run_test(
        "DELETE", "Delete non-existent task", "DELETE", "/tasks/9999", 404,
        lambda: requests.delete(f"{BASE_URL}/tasks/9999")
    )

    # Test 4: Invalid ID
    runner.run_test(
        "DELETE", "Invalid task ID", "DELETE", "/tasks/abc", 400,
        lambda: requests.delete(f"{BASE_URL}/tasks/abc")
    )


def main():
    """Run all API tests."""
    # Check for verbose flag
    verbose = "-v" in sys.argv or "--verbose" in sys.argv

    print("="*120)
    print("🚀 TASK MANAGEMENT API - TEST SUITE")
    print("="*120)
    print(f"Base URL: {BASE_URL}")
    print(f"Mode: {'Verbose' if verbose else 'Summary'}")
    if not HAS_TABULATE:
        print("Note: Install 'tabulate' for better table formatting: pip install tabulate")
    print("="*120)

    # Test connection
    try:
        requests.get(f"{BASE_URL}/tasks", timeout=2)
        print("✅ Server is reachable")
    except requests.exceptions.ConnectionError:
        print("\n❌ ERROR: Cannot connect to API server")
        print(f"Please ensure the server is running on {BASE_URL}")
        print("\nStart the server with:")
        print("  go run ./cmd/api")
        print("  OR")
        print("  docker-compose up")
        return

    # Initialize test runner
    runner = TestRunner(verbose=verbose)

    # Run all tests
    print("\n🧪 Running tests...\n" if not verbose else "")

    task1_id, task2_id = test_create_tasks(runner)

    if task1_id and task2_id:
        test_list_tasks(runner)
        test_get_task(runner, task1_id)
        test_update_task(runner, task1_id)
        test_delete_task(runner, task2_id)
    else:
        print("⚠️  Skipping remaining tests due to task creation failure")

    # Print summary
    runner.print_summary()

    # Exit with appropriate code
    sys.exit(0 if all(r.passed for r in runner.results) else 1)


if __name__ == "__main__":
    main()
