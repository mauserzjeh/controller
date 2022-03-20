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

import "testing"

func TestNewController(t *testing.T) {
	c1 := New(0)
	if c1 == nil {
		t.Errorf("Controller is not initialized: %v", c1)
	}

	c2 := New(1)
	if c2 == nil {
		t.Errorf("Controller is not initialized: %v", c2)
	}

}

func TestGetWorkers(t *testing.T) {
	want := uint(0)
	c1 := New(want)
	got := c1.GetWorkers()

	if int(want) != got {
		t.Errorf("Wrong amount of workers: %v != %v", want, got)
	}

	want = 10
	c2 := New(want)
	got = c2.GetWorkers()

	if int(want) != got {
		t.Errorf("Wrong amount of workers: %v != %v", want, got)
	}
}
