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
	"testing"
	"time"
)

// Helpers ----------------------------------------------------------

// assertEqual fails if the two values are not equal
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got: %v != want: %v", got, want)
	}
}

// assertNotEqual fails if the two values are equal
func assertNotEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got == want {
		t.Errorf("didn't want %v", got)
	}
}

// taskFunc simulates a task that can be given to the controller. It is
// It simply multiplies the given number by 2.
// Duration parameter sets how long the function takes to process (ms).
// Fail parameter sets if the result should end up with an error.
func taskFunc(i, duration int64, fail bool) Result {
	if duration > 0 {
		time.Sleep(time.Duration(duration) * time.Millisecond)
	}

	if fail {
		return Result{
			Output: nil,
			Err:    errors.New("taskCallback failed"),
		}
	}

	return Result{
		Output: i * 2,
		Err:    nil,
	}

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

func TestRestart(t *testing.T) {
	c := New(1)
	c.Stop(true)
	err := c.Restart()
	assertEqual(t, err == nil, true)

	err = c.Restart()
	assertEqual(t, err == ErrControllerMustBeStoppedOrTerminated, true)
}

func TestAddRequest(t *testing.T) {
	c := New(1)
	defer c.Stop(true)

	r := Request{
		TaskFunc: func() Result {
			return taskFunc(1, 0, false)
		},
	}

	res := c.AddRequest(r)
	assertEqual(t, res.Err == nil, true)
	assertEqual(t, res.Output.(int64), 2)
}

func TestAddRequestTimeout(t *testing.T) {
	c := New(1)
	defer c.Stop(true)

	r := Request{
		TaskFunc: func() Result {
			return taskFunc(1, 500, false)
		},
		Timeout: 100 * time.Millisecond,
	}

	res := c.AddRequest(r)
	assertEqual(t, res.Err == ErrReqTimedOut, true)
	assertEqual(t, res.Output == nil, true)
}
