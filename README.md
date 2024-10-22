# gorkerr

A generic worker pool in Go with error handling and concurrency control.

## Overview

`gorkerr` is a Go library that provides a generic worker pool implementation with built-in error handling. It allows you to process jobs concurrently using a pool of worker goroutines. The worker pool can handle any data type thanks to Go's generics, and it stops processing when an error occurs, ensuring robust error management.

## Features

- **Generic Worker Pool**: Supports any job type using Go generics.
- **Concurrency Control**: Specify the number of worker goroutines.
- **Error Handling**: Stops processing when a worker returns an error.
- **Panic Recovery**: Recovers from panics within worker functions.
- **Graceful Shutdown**: Provides a `StopAndWait` method to stop the pool and wait for all jobs to finish.
- **Thread-Safe Feeding**: Safely feed jobs into the pool from multiple goroutines.

## Installation

To install `gorkerr`, use `go get`:

```bash
go get github.com/vlasashk/gorkerr
```

## Usage

Here's how to use `gorkerr` in your project:

```go
package main

import (
    "fmt"
    "github.com/vlasashk/gorkerr"
)

func main() {
    // Define the function to process each job
    workerFunc := func(job int) error {
        fmt.Printf("Processing job: %d\n", job)
        // Simulate error on a specific condition
        if job == 5 {
            return fmt.Errorf("error processing job %d", job)
        }
        return nil
    }

    // Create a new worker pool with 5 workers
    wp := gorkerr.NewWorkerPoolWithErr[int](5, workerFunc)

    // Start the worker pool
    wp.Start()

    // Feed jobs into the pool
    for i := 0; i < 10; i++ {
        wp.Feed(i)
    }

    // Stop the pool and wait for all jobs to finish
    if err := wp.StopAndWait(); err != nil {
        fmt.Printf("Worker pool stopped with error: %v\n", err)
    } else {
        fmt.Println("All jobs processed successfully.")
    }
}
```

**Output:**
```
Processing job: 0
Processing job: 2
Processing job: 1
Processing job: 5
Processing job: 4
Processing job: 3
Worker pool stopped with error: error processing job 5
```

## API Reference

### NewWorkerPoolWithErr

```go
func NewWorkerPoolWithErr[T any](workers int, fn func(T) error) *WorkerPoolWithErr[T]
```

Creates a new worker pool with the specified number of workers and the function to process each job.

- `workers`: Number of worker goroutines to spawn.
- `fn`: Function that processes each job of type `T`. Should return an error if processing fails.

### Start

```go
func (wp *WorkerPoolWithErr[T]) Start()
```

Starts the worker pool by spawning the worker goroutines. Should be called before feeding jobs into the pool.

### Feed

```go
func (wp *WorkerPoolWithErr[T]) Feed(payload T)
```

Adds a job to the queue to be processed by the workers. Can be called concurrently from multiple goroutines.

### StopAndWait

```go
func (wp *WorkerPoolWithErr[T]) StopAndWait() error
```

Stops the worker pool and waits until all workers have finished processing the remaining jobs. Returns an error if any worker returned an error or panicked.

## Error Handling

- If any worker function returns an error, the worker pool will stop processing new jobs.
- `StopAndWait` will return the first error encountered.
- Panics within worker functions are recovered and returned as errors.

## Concurrency Considerations

- The `Feed` method is thread-safe and can be called from multiple goroutines.
- The worker pool uses channels and atomic operations to ensure safe concurrent access.

## Internal Structure

The worker pool is implemented using the following components:

- **Workers**: Goroutines that process jobs from the job queue.
- **Job Queue**: A buffered channel that holds jobs to be processed.
- **Error Group**: Manages the lifetimes of worker goroutines and collects errors.
- **Atomic Flags**: Used for managing the state of the pool (started, closed).

## Example: Processing Strings

```go
package main

import (
    "fmt"
    "strings"
    "github.com/yourusername/gorkerr"
)

func main() {
    workerFunc := func(s string) error {
        fmt.Println(strings.ToUpper(s))
        return nil
    }

    wp := gorkerr.NewWorkerPoolWithErr[string](3, workerFunc)
    wp.Start()

    words := []string{"go", "is", "awesome"}
    for _, word := range words {
        wp.Feed(word)
    }

    if err := wp.StopAndWait(); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

**Output:**
```
GO
IS
AWESOME
```

## Testing

Extensive tests are provided to ensure the robustness of the worker pool under various conditions, including error handling, concurrent feeding, and panic recovery.

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Write tests to cover your changes.
4. Open a pull request with a detailed description of your changes.

---