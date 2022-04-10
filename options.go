package controller

type (
	// option function
	option func(opts *options)

	// options struct holds the configurable options
	options struct {
		workers   int // The amount of workers
		queueSize int // Size of the queue
	}
)

// setOptions sets the given options
func setOptions(opts ...option) *options {
	o := options{}
	for _, option := range opts {
		option(&o)
	}
	return &o
}

// Workers sets the amount of workers
func Workers(amount int) option {
	return func(opts *options) {
		opts.workers = amount
	}
}

// BufferedQueue makes the queue buffered. Setting 0 as bufferSize will
// make the queue unbuffered. Which is equal to not setting this option
func BufferedQueue(bufferSize int) option {
	return func(opts *options) {
		opts.queueSize = bufferSize
	}
}
