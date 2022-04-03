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

// taskFunc simulates a task. It simply multiplies the given number by 2.
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
	defer c.Terminate()

	assertNotEqual(t, c, nil)
}

func TestGetWorkers(t *testing.T) {
	workers := 5
	c := New(workers)
	defer c.Terminate()

	assertEqual(t, c.GetWorkers(), workers)
}

func TestSetWorkers(t *testing.T) {
	workers := 5
	c := New(workers)
	defer c.Terminate()

	assertEqual(t, c.GetWorkers(), workers)

	workers = 10
	c.SetWorkers(workers)
	assertEqual(t, c.GetWorkers(), workers)

	workers = 3
	c.SetWorkers(workers)
	assertEqual(t, c.GetWorkers(), workers)
}

func TestIsRunning(t *testing.T) {
	workers := 1
	c := New(workers)

	assertEqual(t, c.IsRunning(), true)
	c.Terminate()
	assertEqual(t, c.IsRunning(), false)
}

func TestIsTerminated(t *testing.T) {
	workers := 1
	c := New(workers)
	assertEqual(t, c.IsRunning(), true)
	c.Terminate()
	assertEqual(t, c.IsRunning(), false)
	assertEqual(t, c.IsTerminated(), true)
}
