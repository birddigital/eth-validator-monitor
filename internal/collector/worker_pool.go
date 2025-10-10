package collector

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages a pool of goroutines for validator data collection
type WorkerPool struct {
	workers       int
	taskQueue     chan Task
	resultQueue   chan Result
	errorQueue    chan error
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc

	// Metrics
	tasksProcessed atomic.Uint64
	tasksFailed    atomic.Uint64
	activeWorkers  atomic.Int32

	// Configuration
	maxRetries     int
	retryDelay     time.Duration
	taskTimeout    time.Duration
}

// Task represents a validator data collection task
type Task struct {
	ID             string
	ValidatorIndex int64
	Type           TaskType
	Priority       int
	Deadline       time.Time
	Metadata       map[string]interface{}
}

// TaskType defines the type of collection task
type TaskType string

const (
	TaskTypeSnapshot     TaskType = "snapshot"
	TaskTypeBalance      TaskType = "balance"
	TaskTypeAttestation  TaskType = "attestation"
	TaskTypeProposal     TaskType = "proposal"
	TaskTypeSyncCommittee TaskType = "sync_committee"
)

// Result represents the result of a collection task
type Result struct {
	TaskID         string
	ValidatorIndex int64
	Type           TaskType
	Data           interface{}
	CollectedAt    time.Time
	Duration       time.Duration
	Error          error
}

// WorkerPoolConfig contains configuration for the worker pool
type WorkerPoolConfig struct {
	Workers        int
	QueueSize      int
	MaxRetries     int
	RetryDelay     time.Duration
	TaskTimeout    time.Duration
}

// DefaultWorkerPoolConfig returns default configuration
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		Workers:        10,
		QueueSize:      1000,
		MaxRetries:     3,
		RetryDelay:     time.Second * 2,
		TaskTimeout:    time.Second * 30,
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, config *WorkerPoolConfig) *WorkerPool {
	poolCtx, cancel := context.WithCancel(ctx)

	return &WorkerPool{
		workers:       config.Workers,
		taskQueue:     make(chan Task, config.QueueSize),
		resultQueue:   make(chan Result, config.QueueSize),
		errorQueue:    make(chan error, config.Workers),
		ctx:           poolCtx,
		cancel:        cancel,
		maxRetries:    config.MaxRetries,
		retryDelay:    config.RetryDelay,
		taskTimeout:   config.TaskTimeout,
	}
}

// Start initializes and starts all workers
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the main worker goroutine
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	p.activeWorkers.Add(1)
	defer p.activeWorkers.Add(-1)

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}

			// Process task with timeout
			taskCtx, cancel := context.WithTimeout(p.ctx, p.taskTimeout)
			result := p.processTask(taskCtx, task)
			cancel()

			// Send result
			select {
			case p.resultQueue <- result:
			case <-p.ctx.Done():
				return
			}

			// Update metrics
			if result.Error != nil {
				p.tasksFailed.Add(1)
			} else {
				p.tasksProcessed.Add(1)
			}
		}
	}
}

// processTask processes a single task with retry logic
func (p *WorkerPool) processTask(ctx context.Context, task Task) Result {
	startTime := time.Now()
	var lastError error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return Result{
					TaskID:         task.ID,
					ValidatorIndex: task.ValidatorIndex,
					Type:           task.Type,
					Error:          ctx.Err(),
					CollectedAt:    time.Now(),
					Duration:       time.Since(startTime),
				}
			case <-time.After(p.retryDelay * time.Duration(attempt)):
			}
		}

		// Execute task (this will be implemented by specific collectors)
		data, err := p.executeTask(ctx, task)
		if err == nil {
			return Result{
				TaskID:         task.ID,
				ValidatorIndex: task.ValidatorIndex,
				Type:           task.Type,
				Data:           data,
				CollectedAt:    time.Now(),
				Duration:       time.Since(startTime),
				Error:          nil,
			}
		}

		lastError = err

		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
	}

	return Result{
		TaskID:         task.ID,
		ValidatorIndex: task.ValidatorIndex,
		Type:           task.Type,
		Error:          fmt.Errorf("task failed after %d attempts: %w", p.maxRetries+1, lastError),
		CollectedAt:    time.Now(),
		Duration:       time.Since(startTime),
	}
}

// executeTask executes the actual collection task
// This is a placeholder that will be implemented by specific task handlers
func (p *WorkerPool) executeTask(ctx context.Context, task Task) (interface{}, error) {
	// This will be replaced with actual implementation
	// that calls the appropriate collector based on task type
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Millisecond * 100): // Simulate work
		return map[string]interface{}{
			"validator_index": task.ValidatorIndex,
			"collected_at":    time.Now(),
		}, nil
	}
}

// Submit adds a task to the queue
func (p *WorkerPool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWithPriority adds a high-priority task
func (p *WorkerPool) SubmitWithPriority(task Task) error {
	task.Priority = 1 // High priority
	// In a real implementation, we'd have a priority queue
	return p.Submit(task)
}

// Results returns the result channel
func (p *WorkerPool) Results() <-chan Result {
	return p.resultQueue
}

// Errors returns the error channel
func (p *WorkerPool) Errors() <-chan error {
	return p.errorQueue
}

// Shutdown gracefully shuts down the worker pool
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
	// Stop accepting new tasks
	close(p.taskQueue)

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers finished
		p.cancel()
		close(p.resultQueue)
		close(p.errorQueue)
		return nil
	case <-time.After(timeout):
		// Timeout - force shutdown
		p.cancel()
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// Stats returns current pool statistics
func (p *WorkerPool) Stats() PoolStats {
	return PoolStats{
		TasksProcessed: p.tasksProcessed.Load(),
		TasksFailed:    p.tasksFailed.Load(),
		ActiveWorkers:  p.activeWorkers.Load(),
		QueueSize:      len(p.taskQueue),
		ResultQueueSize: len(p.resultQueue),
	}
}

// PoolStats contains worker pool statistics
type PoolStats struct {
	TasksProcessed  uint64
	TasksFailed     uint64
	ActiveWorkers   int32
	QueueSize       int
	ResultQueueSize int
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// Implement logic to determine retryable errors
	// For now, we'll consider timeouts and temporary network errors as retryable
	if err == context.DeadlineExceeded {
		return true
	}
	// Add more conditions as needed
	return false
}