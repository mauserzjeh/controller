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
	"testing"
	"time"
)

// Helpers ----------------------------------------------------------

// assertEqual fails if the two values are not equal
func assertEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got: %v != want: %v", got, want)
	}
}

// assertNotEqual fails if the two values are equal
func assertNotEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got == want {
		t.Errorf("didn't want %v", got)
	}
}

// taskfn simulates a task that can be given to the controller. It is
// It simply multiplies the given number by 2.
// Duration parameter sets how long the function takes to process (ms).
// Fail parameter sets if the result should end up with an error.
func taskfn(i, duration int64, fail bool) (any, error) {
	if duration > 0 {
		time.Sleep(time.Duration(duration) * time.Millisecond)
	}

	if fail {
		return nil, errors.New("taskfn failed")
	}

	return i * 2, nil
}

// Tests ------------------------------------------------------------

func TestNewController(t *testing.T) {
	c := New(0)
	defer c.Stop(true)

	assertNotEqual(t, c, nil)
}

func TestGetWorkers(t *testing.T) {
	workers := 5
	c := New(workers)
	defer c.Stop(true)

	assertEqual(t, c.GetWorkers(), workers)

	c2 := New(0)
	defer c2.Stop(true)

	assertEqual(t, c2.GetWorkers(), 1)

	c3 := New(-1)
	defer c3.Stop(true)

	assertEqual(t, c3.GetWorkers(), 1)
}

func TestSetWorkers(t *testing.T) {
	workers := 5
	c := New(workers)
	defer c.Stop(true)

	assertEqual(t, c.GetWorkers(), workers)

	workers = 10
	c.SetWorkers(workers)
	assertEqual(t, c.GetWorkers(), workers)

	workers = 3
	c.SetWorkers(workers)
	assertEqual(t, c.GetWorkers(), workers)

	c.SetWorkers(workers)
	assertEqual(t, c.GetWorkers(), workers)
}

func TestSetWorkersWhileProcessingTasks(t *testing.T) {
	c := New(1)
	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 500, false)
		},
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := c.AddRequest(req)
			assertEqual(t, res.(int64), 2)
			assertEqual(t, err == nil, true)
		}()
	}

	time.Sleep(250 * time.Millisecond)
	c.SetWorkers(10)
	assertEqual(t, c.GetWorkers(), 10)

	time.Sleep(250 * time.Millisecond)
	c.SetWorkers(2)
	assertEqual(t, c.GetWorkers(), 2)

	wg.Wait()
	c.Stop(true)
}

func TestIsRunning(t *testing.T) {
	c := New(1)
	assertEqual(t, c.IsRunning(), true)
	c.Stop(true)
	assertEqual(t, c.IsRunning(), false)
}

func TestIsStopped(t *testing.T) {
	c := New(1)
	c.Stop(true)
	assertEqual(t, c.IsRunning(), false)
	assertEqual(t, c.IsStopped(), true)
}

func TestIsShuttingDown(t *testing.T) {
	c := New(1)
	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 1000, false)
		},
	}

	res := c.AddAsyncRequest(req)
	time.Sleep(250 * time.Millisecond)
	go c.Stop(false)
	time.Sleep(250 * time.Millisecond)
	assertEqual(t, c.IsShuttingDown(), true)
	r := <-res
	assertEqual(t, r.Err == nil, true)
	assertEqual(t, r.Output.(int64), 2)
}

func TestRestart(t *testing.T) {
	c := New(1)
	c.Stop(true)
	err := c.Restart()
	assertEqual(t, err == nil, true)

	err = c.Restart()
	assertEqual(t, err == ErrControllerMustBeStopped, true)
}

func TestStop(t *testing.T) {
	c := New(1)
	err := c.Stop(true)
	assertEqual(t, err == nil, true)
	assertEqual(t, c.IsStopped(), true)

	err = c.Stop(true)
	assertEqual(t, err == ErrControllerIsNotRunning, true)
}

func TestStopLongRunningTasks(t *testing.T) {
	t.Run("stop", func(t *testing.T) {
		c := New(1)

		req := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 1000, false)
			},
		}

		for i := 0; i < 3; i++ {
			go func() {
				res, err := c.AddRequest(req)
				assertEqual(t, err == nil, true)
				assertEqual(t, res.(int64), 2)
			}()
		}
		time.Sleep(250 * time.Millisecond)
		c.Stop(false)
	})

	t.Run("terminate", func(t *testing.T) {
		c := New(1)

		req := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 1000, false)
			},
		}

		for i := 0; i < 3; i++ {
			go func() {
				res, err := c.AddRequest(req)
				assertEqual(t, err == ErrReqInterrupted || err == ErrControllerIsNotRunning, true)
				assertEqual(t, res == nil, true)
			}()
		}

		time.Sleep(250 * time.Millisecond)
		c.Stop(true)
	})
}

func TestAddRequest(t *testing.T) {
	c := New(1)
	defer c.Stop(true)

	r := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 0, false)
		},
	}

	res, err := c.AddRequest(r)
	assertEqual(t, err == nil, true)
	assertEqual(t, res.(int64), 2)
}

func TestAddRequestStopped(t *testing.T) {
	c := New(1)
	c.Stop(true)

	r := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 0, false)
		},
	}

	res, err := c.AddRequest(r)
	assertEqual(t, err == ErrControllerIsNotRunning, true)
	assertEqual(t, res == nil, true)
}

func TestAddRequestTimeout(t *testing.T) {

	t.Run("function_times_out", func(t *testing.T) {
		c := New(1)
		defer c.Stop(true)

		r := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 200, false)
			},
			Timeout: 100 * time.Millisecond,
		}

		res, err := c.AddRequest(r)
		assertEqual(t, err == ErrReqTimedOut, true)
		assertEqual(t, res == nil, true)
	})

	t.Run("function_does_not_timeout", func(t *testing.T) {
		c := New(1)
		defer c.Stop(true)

		r := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 0, false)
			},
			Timeout: 100 * time.Millisecond,
		}

		res, err := c.AddRequest(r)
		assertEqual(t, err == nil, true)
		assertEqual(t, res.(int64), 2)
	})
}

func TestAddAsyncRequest(t *testing.T) {
	c := New(1)
	defer c.Stop(true)

	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 0, false)
		},
	}

	res := c.AddAsyncRequest(req)
	r := <-res
	assertEqual(t, r.Err == nil, true)
	assertEqual(t, r.Output.(int64), 2)
}

func TestAddAsyncRequestTimeout(t *testing.T) {

	t.Run("function_times_out", func(t *testing.T) {
		c := New(1)
		defer c.Stop(true)

		req := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 200, false)
			},
			Timeout: 100 * time.Millisecond,
		}

		res := c.AddAsyncRequest(req)
		r := <-res
		assertEqual(t, r.Err == ErrReqTimedOut, true)
		assertEqual(t, r.Output == nil, true)
	})

	t.Run("function_does_not_timeout", func(t *testing.T) {
		c := New(1)
		defer c.Stop(true)

		req := Request{
			TaskFunc: func() (any, error) {
				return taskfn(1, 0, false)
			},
			Timeout: 100 * time.Millisecond,
		}

		res := c.AddAsyncRequest(req)
		r := <-res
		assertEqual(t, r.Err == nil, true)
		assertEqual(t, r.Output.(int64), 2)
	})
}

func TestQueuedTasks(t *testing.T) {
	c := New(1)

	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 1000, false)
		},
	}

	for i := 0; i < 5; i++ {
		c.AddAsyncRequest(req)
	}

	assertNotEqual(t, c.QueuedTasks(), 0)
	c.Stop(true)
}

func TestControllerLoop(t *testing.T) {
	c := New(1)
	c.loop()
	c.Stop(true)
	assertEqual(t, c.IsStopped(), true)
}

// Benchmarks -------------------------------------------------------

const (
	benchmarkWorkers  = 2000
	benchmarkRequests = 1000000
)

func BenchmarkRequest(b *testing.B) {
	c := New(benchmarkWorkers)

	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 0, false)
		},
	}

	for i := 0; i < b.N; i++ {
		for j := 0; j < benchmarkRequests; j++ {
			_ = c.AddAsyncRequest(req)
		}
	}

	c.Stop(false)
}

func BenchmarkRequestTimeout(b *testing.B) {
	c := New(benchmarkWorkers)

	req := Request{
		TaskFunc: func() (any, error) {
			return taskfn(1, 10, false)
		},
		Timeout: 1 * time.Second,
	}

	for i := 0; i < b.N; i++ {
		for j := 0; j < benchmarkRequests; j++ {
			_ = c.AddAsyncRequest(req)
		}
	}

	c.Stop(false)
}
