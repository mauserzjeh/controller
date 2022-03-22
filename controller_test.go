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

// taskCallback simulates a callback. Duration parameter sets how long the function
// takes to process (ms). Fail parameter sets if the result should end up with an error
func taskCallback(i, duration int64, fail bool) Result {
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
	assertNotEqual(t, c, nil)
}

func TestGetWorkers(t *testing.T) {
	want := uint(0)
	c := New(want)
	got := c.GetWorkers()

	assertEqual(t, got, int(want))
}

func TestSetWorkers(t *testing.T) {
	want := uint(0)
	c := New(want)
	got := c.GetWorkers()
	assertEqual(t, got, int(want))

	want = uint(10)
	c.SetWorkers(want)
	got = c.GetWorkers()
	assertEqual(t, got, int(want))

	c.Start()

	want = uint(5)
	c.SetWorkers(want)
	got = c.GetWorkers()
	assertEqual(t, got, int(want))

	want = uint(10)
	c.SetWorkers(want)
	got = c.GetWorkers()
	assertEqual(t, got, int(want))

	c.Stop()
}

func TestIsRunning(t *testing.T) {
	c := New(10)
	assertEqual(t, c.IsRunning(), false)

	c.Start()
	assertEqual(t, c.IsRunning(), true)

	c.Stop()
	assertEqual(t, c.IsRunning(), false)
}

func TestDoubleStart(t *testing.T) {
	w := 10
	c := New(uint(w))
	assertEqual(t, c.IsRunning(), false)
	assertEqual(t, c.GetWorkers(), w)

	c.Start()
	assertEqual(t, c.IsRunning(), true)
	assertEqual(t, c.GetWorkers(), w)

	c.Start()
	assertEqual(t, c.IsRunning(), true)
	assertEqual(t, c.GetWorkers(), w)

	c.Stop()
	assertEqual(t, c.IsRunning(), false)
}
