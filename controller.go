// MIT License
//
// Copyright (c) 2022 Soma Rádóczi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package controller

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	stateRunning = iota
	stateShutDown
	stateStopped
	stateTerminated
)

var (
	ErrReqTimedOut            = errors.New("request timed out")
	ErrReqInterrupted         = errors.New("request interrupted")
	ErrControllerStopped      = errors.New("controller stopped")
	ErrControllerShuttingDown = errors.New("controller is shutting down")
	ErrControllerTerminated   = errors.New("controller terminated")
	ErrControllerIsNotStopped = errors.New("controller is not stopped")
	ErrControllerIsNotRunning = errors.New("controller is not running")
)

type (

	// Request struct that can be sent to the controller
	Request struct {
		TaskFunc func() Result // A function to do
		Timeout  time.Duration // Optional timeout
	}

	// Result struct will hold the result of a request
	Result struct {
		Output any   // Output of the task function
		Err    error // Error of the task function
	}

	// controller struct represents the controller
	controller struct {
		pool        chan chan task // Pool for workers
		queue       chan task      // Queue for queued tasks
		queuedTasks int32          // Counter for queued tasks
		workers     []*worker      // Workers that the controller holds
		freeWorkers int32          // The amount of workers in the pool
		state       int32          // State of the controller
		cQuit       chan struct{}  // Quit signal channel
		cExited     chan struct{}  // Exited signal channel
		mux         sync.Mutex     // Mutex for locking
	}

	// worker struct represents a single worker
	worker struct {
		pool       chan<- chan task // Pool where the worker registers itself
		w          chan task        // The request channel of the worker where it will receive a task after registration
		cStop      chan struct{}    // Stop signal channel
		cTerminate chan struct{}    // Terminate signal channel
		cExited    chan struct{}    // Exited signal channel
	}

	// task struct represents a task
	task struct {
		taskFunc func() Result // A function to do
		result   chan Result   // Result channel where the result will be sent back
	}
)

// ------------------------------------------------------------------
// Controller
// ------------------------------------------------------------------

// New creates a new controller instance with a set amount of workers
func New(workers int) *controller {
	c := controller{
		pool:    make(chan chan task),
		queue:   make(chan task),
		state:   stateStopped,
		cQuit:   make(chan struct{}),
		cExited: make(chan struct{}),
	}

	c.SetWorkers(workers) // Initialize workers
	c.loop()              // Start controller loop
	return &c
}

// loop runs a controller loop
func (c *controller) loop() {

	// Avoid starting multiple loops
	if atomic.LoadInt32(&c.state) != stateStopped {
		return
	}

	// Set state to running
	atomic.StoreInt32(&c.state, stateRunning)

	go func() {

		defer func() {
			close(c.cExited)
		}()

		for {
			select {
			case t := <-c.queue:
				// Receive a task from the queue
				select {
				case w := <-c.pool:
					// Get a worker from the pool and assign the task to the worker
					w <- t
					atomic.AddInt32(&c.queuedTasks, -1)

				case <-c.cQuit:
					// Receive quit signal
					return
				}

			case <-c.cQuit:
				// Receive quit signal
				return
			}
		}
	}()
}

// GetWorkers returns the amount of workers the controller controlls
func (c *controller) GetWorkers() int {
	c.mux.Lock()
	defer c.mux.Unlock()

	return len(c.workers)
}

// SetWorkers sets the amount of workers the controller controlls.
// If the size is reduced, then the stopped workers will be removed from the pool.
// If a worker is stopped in the middle of a process, then the goroutine will block
// until the process is finished. Otherwise it will stop immediately
// If the size is increased, then new workers will be added to the pool.
func (c *controller) SetWorkers(workers int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	// Don't do anything if the size  wasn't changed
	lw := len(c.workers)
	if lw == workers {
		return
	}

	// There must be at least 1 worker in the pool
	if workers <= 0 {
		workers = 1
	}

	// If the size was increased, then add new workers
	for i := lw; i < workers; i++ {
		c.workers = append(c.workers, newWorker(c.pool))
	}

	// If the size was decreased, then stop the workers that will be deleted.
	// Stopping happens asynchronously
	for i := workers; i < lw; i++ {
		c.workers[i].stop()
	}

	// Wait for all stopped workers to finish
	for i := workers; i < lw; i++ {
		c.workers[i].exited()
		c.workers[i] = nil
	}

	// Remove unused workers from the pool
	c.workers = c.workers[:workers]
}

func (c *controller) stopWorkers(terminate bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	lw := len(c.workers)

	if terminate {
		// terminate workers
		for i := 0; i < lw; i++ {
			c.workers[i].terminate()
		}

	} else {
		// stop workers
		for i := 0; i < lw; i++ {
			c.workers[i].stop()
		}
	}

	for i := 0; i < lw; i++ {
		c.workers[i].exited()
		c.workers[i] = nil
	}
}

// IsRunning returns the state of the controller
func (c *controller) IsRunning() bool {
	return atomic.LoadInt32(&c.state) == stateRunning
}

// IsTerminated returns if the controller was terminated and cannot be restarted anymore
func (c *controller) IsTerminated() bool {
	return atomic.LoadInt32(&c.state) == stateTerminated
}

// IsShuttingDown returns if the controller is shutting down
func (c *controller) IsShuttingDown() bool {
	return atomic.LoadInt32(&c.state) == stateShutDown
}

// IsStopped returns if the controller is stopped
func (c *controller) IsStopped() bool {
	return atomic.LoadInt32(&c.state) == stateStopped
}

// QueuedTasks returns the amount of tasks in the queue
func (c *controller) QueuedTasks() int32 {
	return atomic.LoadInt32(&c.queuedTasks)
}

// Stop stops the controller and the processing of the queue
// All tasks that have been in progress will be finished. The function blocks until
// the controller is not stopped completely.
func (c *controller) Stop(purgeQueue bool) error {
	if !atomic.CompareAndSwapInt32(&c.state, stateRunning, stateShutDown) {
		return ErrControllerIsNotRunning
	}

	close(c.cQuit)
	c.stopWorkers(false)

	<-c.cExited

	// Purge the queue by reseting the channel
	if purgeQueue {
		close(c.queue)
		c.queue = make(chan task)
	}

	atomic.SwapInt32(&c.state, stateStopped)
	return nil
}

// Terminate terminates the controller. All tasks that have been in progress will be terminated.
// The function blocks until the controller is not terminated completely. The controller
// will not be able to restarted after this function is called.
func (c *controller) Terminate() error {
	if !atomic.CompareAndSwapInt32(&c.state, stateRunning, stateShutDown) {
		return ErrControllerIsNotRunning
	}

	close(c.cQuit)
	c.stopWorkers(true)

	<-c.cExited

	// Purge the queue
	close(c.queue)
	c.queue = nil

	atomic.SwapInt32(&c.state, stateTerminated)
	return nil
}

// Restart tries to restart the controller. If the queue was not purged when the controller
// was stopped, then the queued tasks will be processed as well.
func (c *controller) Restart() error {
	if atomic.LoadInt32(&c.state) == stateTerminated {
		return ErrControllerTerminated
	}

	if !atomic.CompareAndSwapInt32(&c.state, stateStopped, stateRunning) {
		return ErrControllerIsNotStopped
	}

	c.cQuit = make(chan struct{})
	c.cExited = make(chan struct{})

	c.mux.Lock()
	for i := 0; i < len(c.workers); i++ {
		c.workers[i] = newWorker(c.pool)
	}
	c.mux.Unlock()

	c.loop()
	return nil
}

// AddRequest adds a new request to the queue
// and returns the result when it is processed.
// The function blocks until the result is returned
func (c *controller) AddRequest(r Request) Result {
	return c.addTask(r)
}

// AddAsyncRequest adds a new request to the queue and returns
// a channel on which the result can be received.
// The function does not block.
func (c *controller) AddAsyncRequest(r Request) chan Result {
	res := make(chan Result, 1)

	go func() {
		res <- c.addTask(r)
		close(res)
	}()

	return res
}

// addTask adds a new task to the queue and returns the result
func (c *controller) addTask(r Request) Result {
	if !c.IsRunning() {
		return Result{
			Output: nil,
			Err:    ErrControllerIsNotRunning,
		}
	}

	t := task{
		taskFunc: r.taskFuncTimeoutWrapper(),
		result:   make(chan Result, 1),
	}

	select {

	// If a worker is available then give the task to the worker
	case w := <-c.pool:
		w <- t

	// Otherwise put the task to the queue
	case c.queue <- t:
		atomic.AddInt32(&c.queuedTasks, 1)
	}

	// Return the result
	return <-t.result
}

// taskFuncTimeoutWrapper
func (r *Request) taskFuncTimeoutWrapper() func() Result {
	if r.Timeout <= 0 {
		return r.TaskFunc
	}

	return func() Result {
		res := make(chan Result, 1)

		timeout := time.NewTimer(r.Timeout)
		select {
		case res <- r.TaskFunc():
			timeout.Stop()
			return <-res

		case <-timeout.C:
			return Result{
				Output: nil,
				Err:    ErrReqTimedOut,
			}
		}
	}
}

// ------------------------------------------------------------------
// Worker
// ------------------------------------------------------------------

// newWorker creates a new worker instance
func newWorker(pool chan<- chan task) *worker {
	w := worker{
		pool:       pool,
		w:          make(chan task, 1),
		cStop:      make(chan struct{}),
		cTerminate: make(chan struct{}),
		cExited:    make(chan struct{}),
	}

	go w.loop() // Start worker loop
	return &w
}

// loop runs a worker loop
func (w *worker) loop() {
	defer func() {
		close(w.cExited)
	}()

	for {
		select {
		case w.pool <- w.w:
			// Register the worker to the pool

			select {
			case task := <-w.w:
				// Worker received a task
				select {
				case task.result <- task.taskFunc():
					// Process the task and send back the result
					close(task.result)

				case <-w.cTerminate:
					// Worker got terminated while the task was still processing
					task.result <- Result{
						Output: nil,
						Err:    ErrReqInterrupted,
					}

					close(task.result)
					return
				}

			case <-w.cStop:
				// Receive stop signal
				return

			case <-w.cTerminate:
				// Receive terminate signal
				return
			}

		case <-w.cStop:
			// Receive stop signal
			return

		case <-w.cTerminate:
			// Receive terminate signal
			return
		}
	}
}

// stop closes the stop channel signaling the worker to stop
func (w *worker) stop() {
	close(w.cStop)
}

// terminate closes the terminate channel signaling the worker to terminate
func (w *worker) terminate() {
	close(w.cTerminate)
}

// exited blocks until the worker has exited from its loop
func (w *worker) exited() {
	<-w.cExited
}
