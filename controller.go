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
	"time"
)

var (
	ErrReqTimedOut      = errors.New("request timed out")
	ErrReqInterrupted   = errors.New("request interrupted")
	ErrControlleStopped = errors.New("controller stopped")
)

type (
	Request struct {
		Callback func() Result
		Timeout  time.Duration
	}

	Result struct {
		Output any
		Err    error
	}
	controller struct {
		requests  chan task
		workers   []*worker
		isRunning bool
		mux       sync.Mutex
	}

	worker struct {
		tasks     <-chan task
		stopC     chan struct{}
		killC     chan struct{}
		stoppedC  chan struct{}
		isRunning bool
		mux       sync.Mutex
	}

	task struct {
		callback func() Result
		timeout  time.Duration
		result   chan Result
	}
)

// ------------------------------------------------------------------
// Controller -------------------------------------------------------
// ------------------------------------------------------------------

// New creates a new controller instance with a set amount of workers
func New(workers uint) *controller {
	c := controller{
		requests:  make(chan task),
		isRunning: false,
	}

	// Initialize workers
	c.SetWorkers(workers)

	return &c
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
// If the size is increased, then new workers will be added to the pool. If the controller
// is already running, then the newly added workers will be started automatically.
func (c *controller) SetWorkers(workers uint) {
	c.mux.Lock()
	defer c.mux.Unlock()

	lw := uint(len(c.workers))

	if lw == workers {
		return
	}

	// If the size was increased, then add new workers
	for i := lw; i < workers; i++ {
		c.workers = append(c.workers, newWorker(c.requests))

		// If the controller is already running, then run the workers too
		if c.isRunning {
			go c.workers[i].start()
		}
	}

	// If the size was decreased, then stop the workers that will be deleted.
	// Stopping happens asynchronously
	for i := workers; i < lw; i++ {
		c.workers[i].stop()
	}

	// Wait for all stopped workers to finish
	for i := workers; i < lw; i++ {
		c.workers[i].stopped()
		c.workers[i] = nil
	}

	// Remove unused workers from the pool
	c.workers = c.workers[:workers]
}

// AddRequest adds a new request to the queue
// and returns the result when it is processed.
// The function blocks until the result is returned
func (c *controller) AddRequest(req Request) Result {

	t := task{
		callback: req.Callback,
		timeout:  req.Timeout,
		result:   make(chan Result, 1),
	}

	return c.addTask(t)
}

// AddAsyncRequest adds a new request to the queue and returns
// a channel on which the result can be received.
// The function does not block.
func (c *controller) AddAsyncRequest(req Request) chan Result {
	res := make(chan Result, 1)

	go func() {
		t := task{
			callback: req.Callback,
			timeout:  req.Timeout,
			result:   make(chan Result, 1),
		}

		res <- c.addTask(t)
		close(res)
	}()

	return res
}

// addTask adds a new task to the request queue and returns the result
func (c *controller) addTask(t task) Result {

	// If the controller is not running return with an error
	if !c.isRunning {
		return Result{
			Output: nil,
			Err:    ErrControlleStopped,
		}
	}

	// Queue the task
	c.requests <- t

	var r Result

	// If timeout was set then process with timeout
	if t.timeout > 0 {
		timeout := time.NewTimer(t.timeout)
		select {
		case r = <-t.result:
		case <-timeout.C:
			r = Result{
				Output: nil,
				Err:    ErrReqTimedOut,
			}
		}

		timeout.Stop()
		return r
	}

	// Otherwise return when ready
	return <-t.result
}

// QueuedRequests returns the amount of requests queued
func (c *controller) QueuedRequests() int {
	c.mux.Lock()
	defer c.mux.Unlock()

	return len(c.requests)
}

// IsRunning returns the state of the controller
func (c *controller) IsRunning() bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.isRunning
}

// Stop stops the controller from receiving new requests.
// The processing of queued requests continues
func (c *controller) Stop() {
	c.mux.Lock()
	defer c.mux.Unlock()

	// Stop the controller, but the workers
	// will keep processing the queued requests
	c.isRunning = false
}

// Kill stops the controller from receiving new requests.
// The queued requests are thrown away
func (c *controller) Kill() {
	c.mux.Lock()
	defer c.mux.Unlock()

	// Stop the controller from receiving new requests
	c.isRunning = false

	// Stop workers, all tasks in progress will be interrupted
	// and an error will be returned
	for i := 0; i < len(c.workers); i++ {
		c.workers[i].kill()
	}

	// Drain channel so if the controller is restarted
	// then old data will not be processed
	for len(c.requests) > 0 {
		<-c.requests
	}

}

// Start starts the workers of the controller and lets the controller
// receive new requests
func (c *controller) Start() {
	c.mux.Lock()
	defer c.mux.Unlock()

	// Avoid starting if the controller is already running
	if c.isRunning {
		return
	}

	// Start the workers
	for i := 0; i < len(c.workers); i++ {
		go c.workers[i].start()
	}

	c.isRunning = true
}

// ------------------------------------------------------------------
// Worker -----------------------------------------------------------
// ------------------------------------------------------------------

// newWorker creates a new worker instance
func newWorker(tasks <-chan task) *worker {
	w := worker{
		tasks:     tasks,
		isRunning: false,
	}

	return &w
}

// start starts a worker loop which processes the incoming tasks
func (w *worker) start() {

	// Lock mutex
	w.mux.Lock()

	// Check if the worker is already running.
	// If it is running then unlock the mutex and return
	if w.isRunning {
		w.mux.Unlock()
		return
	}

	w.stopC = make(chan struct{})    // Set stop signal channel
	w.killC = make(chan struct{})    // Set kill signal channel
	w.stoppedC = make(chan struct{}) // Set stopped signal channel
	w.isRunning = true               // Set running to true

	// Unlock mutex
	w.mux.Unlock()

	// Create a defer function that will set running state to false and
	// also close the stopped signal channel
	defer func() {
		w.mux.Lock()
		defer w.mux.Unlock()

		w.isRunning = false
		close(w.stoppedC)
	}()

	// Worker loop
	for {
		select {
		case task := <-w.tasks:
			// Receive a task if possible

			select {
			case task.result <- task.callback():
				// Process task, send back the result and
				// close the result channel
				close(task.result)

			case <-w.killC:
				// If kill signal is received before the task callback
				// finishes processing, then return with an error
				task.result <- Result{
					Output: nil,
					Err:    ErrReqInterrupted,
				}

				// Close the result channel and quit from worker loop
				close(task.result)
				return
			}

		case <-w.stopC:
			// Quit from worker loop
			return

		case <-w.killC:
			// Quit from worker loop
			return
		}
	}
}

// stop stops the worker loop.
// It will let a task finish if it is in process already
func (w *worker) stop() {
	w.mux.Lock()
	defer w.mux.Unlock()

	if !w.isRunning {
		return
	}
	close(w.stopC)
}

// kill stops the worker loop.
// It will interrupt a task if it is in process already
func (w *worker) kill() {
	w.mux.Lock()
	defer w.mux.Unlock()

	if !w.isRunning {
		return
	}
	close(w.killC)
}

// stopped blocks until the worker is stopped
func (w *worker) stopped() {
	w.mux.Lock()
	defer w.mux.Unlock()

	if !w.isRunning {
		return
	}
	<-w.stoppedC
}
