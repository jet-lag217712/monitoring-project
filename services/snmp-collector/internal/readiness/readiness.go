package readiness

// Checker reports whether the collector process is ready to serve traffic.
// Implementations must never expose secrets in error strings used by HTTP handlers.
type Checker interface {
	Ready() bool
}

// Func adapts a function to Checker.
type Func func() bool

// Ready implements Checker.
func (f Func) Ready() bool {
	return f()
}

// Status aggregates readiness inputs without retaining secret material.
type Status struct {
	HasConfig      bool
	StorageReady   bool
	BufferReady    bool
	PublisherReady bool
}

// Ready reports overall readiness.
func (s Status) Ready() bool {
	return s.HasConfig && s.StorageReady && s.BufferReady && s.PublisherReady
}

// Evaluate builds Status from plain booleans.
func Evaluate(hasConfig, storageReady, bufferReady, publisherReady bool) Status {
	return Status{
		HasConfig:      hasConfig,
		StorageReady:   storageReady,
		BufferReady:    bufferReady,
		PublisherReady: publisherReady,
	}
}
