# Controller

Controller is a worker pool implementation in Go. It can take requests and execute them concurrently with a limited amount of workers.

# Features
- Zero dependencies
- Create a controller with fixed amount of workers
- Use unbuffered / buffered queue for requests
- Change the amount of workers at runtime
- Add requests
    - Normal requests block the current goroutine until the result is returned
    - Async requests will return a channel where the result can be received, thus not blocking the current goroutine
    - Set timeout for the request
- Stop the controller
    - Wait for queued tasks to complete
    - Terminate all queued tasks
- Restart the controller
- Get the state of the controller
- Get the amount of tasks queued

# Motivation
The main motivation of this library was to simply practice Go and get more familiar with goroutines and concurrency. This implementation is probably not the most performant one compared to other famous worker pool libraries, but feel free to run benchmarks.

Here are some of the resources that served as an inspiration:
- http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/
- https://brandur.org/go-worker-pool
- https://gobyexample.com/worker-pools
- https://github.com/panjf2000/ants
- https://github.com/gammazero/workerpool
- https://github.com/alitto/pond
- https://github.com/Jeffail/tunny

# Installation
```
go get -u github.com/mauserzjeh/controller
```
# Tests
```
go test -v
```
# Benchmarks
Benchmarks use 2K workers and 1M requets with a buffered queue of 100K
```
go test -bench=. -benchmem=true -run=none
```
# Examples

### Add request
```golang
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/mauserzjeh/controller"
)

// A function with return value
func fRet(i int) int {
	return 2 * i
}

// A function with an error return value
func fErr(i int) error {
	if i > 1 {
		return errors.New("oops an error!")
	}

	return nil
}

// A function with return value and error
func fRetNErr(i int) (int, error) {
	if i > 1 {
		return 0, errors.New("oops an error!")
	}

	return 2 * i, nil
}

// A function without return value
func fVoid() {
	time.Sleep(100 * time.Millisecond)
}

func main() {

	// Create a controller
	c := controller.New(
		controller.Workers(1000),         // If omitted, then the minimum default workers is 1
		controller.BufferedQueue(100000), // If omitted or 0 is given, then the queue is unbuffered
	)

	reqfRet := controller.Request{
		TaskFunc: func() (any, error) {
			ret := fRet(1)
			return ret, nil
		},
	}
	res, err := c.AddRequest(reqfRet)
	fmt.Printf("reqfRet - res: %v, err: %v\n", res, err)
	// reqfRet - res: 2, err: <nil>

	// --------------------------------------------------------

	for i := 1; i < 3; i++ {
		reqfErr := controller.Request{
			TaskFunc: func() (any, error) {
				err := fErr(i)
				return nil, err
			},
		}
		res, err := c.AddRequest(reqfErr)
		fmt.Printf("reqfErr_%v - res: %v, err: %v\n", i, res, err)
	}

	// reqfErr_1 - res: <nil>, err: <nil>
	// reqfErr_2 - res: <nil>, err: oops an error!

	// --------------------------------------------------------

	for i := 1; i < 3; i++ {
		fRetNErr := controller.Request{
			TaskFunc: func() (any, error) {
				return fRetNErr(i)
			},
		}
		res, err := c.AddRequest(fRetNErr)
		fmt.Printf("fRetNErr_%v - res: %v, err: %v\n", i, res, err)
	}

	// fRetNErr_1 - res: 2, err: <nil>
	// fRetNErr_2 - res: 0, err: oops an error!

	// --------------------------------------------------------

	reqfVoid := controller.Request{
		TaskFunc: func() (any, error) {
			fVoid()
			return nil, nil
		},
	}

	res, err = c.AddRequest(reqfVoid)
	fmt.Printf("reqfVoid - res: %v, err: %v\n", res, err)

	// reqfVoid - res: <nil>, err: <nil>

	reqfVoidTimed := controller.Request{
		TaskFunc: func() (any, error) {
			fVoid()
			return nil, nil
		},
		Timeout: 5 * time.Millisecond,
	}

	res, err = c.AddRequest(reqfVoidTimed)
	fmt.Printf("reqfVoidTimed - res: %v, err: %v\n", res, err)

	// reqfVoidTimed - res: <nil>, err: request timed out

	c.Stop(false)
}
```
### Add async request
```golang
// Result can be received on the returned channel
resultChannel := c.AddAsyncRequest(req)

// ....................
// ... do something ...
// ....................

result := <-resultChannel

// result.Output will hold the returned value if there was any
// result.Err will hold the returned error if there was any. 

// The Err field will hold the error either returned by the function (if it returns any errors)
// or the error assigned by the controller (timeout, terminate error, 
// trying to send request to the controller when it's not running etc...)

```

### Stop or terminate
```golang
terminate := true // or false

 // Stop the controller. If terminate is set to true then all queued tasks will
 // be discarded and return an error. If set to false, then the controller will wait
 // until all the queued tasks finish, blocking the current goroutine.
err := c.Stop(terminate)

// An error value is returned, indicating if the stop was succesful (ie. can't stop the controller if it's shutting down or already stopped)
```

### Adjust workers at runtime
```golang
// The minimum amount of workers is 1. So if a value, less than the default 1 is given, then
// the controller will set its worker amount to 1

c.SetWorkers(100) // Sets the amount of workers to 100

// ....................
// ... do something ...
// ....................

c.SetWorkers(-1) // Sets the amount of workers to default 1
```

### Restart
```golang
// Terminate the controller
c.Stop(true)

// ....................
// ... do something ...
// ....................

// Restart will try to restart the controller. It will return an error value indicating if
// the restart was succesful (ie. can't restart the controller if it's not completely stopped)
err := c.Restart()
```

### Options
```golang
// Create new controller
c := controller.New(
    controller.Workers(1000), // If omitted, then the minimum default workers is 1
    controller.BufferedQueue(100000), // If omitted or 0 is given, then the queue is unbuffered
)
```

### States and queued tasks
```golang
isRunning := c.IsRunning() // true or false
isShuttingDown := c.IsShuttingDown() // true or false
isStopped := c.IsStopped() // true or false
queuedTasks := c.QueuedTasks() // int32
```
