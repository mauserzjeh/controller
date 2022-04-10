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

type (

	// worker struct represents a single worker
	worker struct {
		pool       chan<- chan task // Pool where the worker registers itself
		w          chan task        // The request channel of the worker where it will receive a task after registration
		cStop      chan struct{}    // Stop signal channel
		cTerminate chan struct{}    // Terminate signal channel
		cExited    chan struct{}    // Exited signal channel
	}
)

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
		case <-w.cStop: // Received stop signal
			return
		case <-w.cTerminate: // Received terminate signal
			return

		case w.pool <- w.w: // Register worker to the pool

			task := <-w.w // Received a task
			rchan := make(chan Result, 1)

			go func() {
				rchan <- task.taskFunc()
				close(rchan)
			}()

			select {
			case <-w.cTerminate: // Received terminate signal
				task.result <- Result{
					Output: nil,
					Err:    ErrReqInterrupted,
				}

				close(task.result)
				return

			case res := <-rchan: // Finished task
				task.result <- res
				close(task.result)

			}
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
