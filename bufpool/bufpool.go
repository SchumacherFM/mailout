package bufpool

import (
	"bytes"
	"sync"
)

// AverageBufferSize should be adjusted to the average size of a bytes.buffer
// in your application.
var AverageBufferSize int = 256

var bufferPool = &sync.Pool{
	New: func() interface{} {
		b := bytes.NewBuffer(make([]byte, AverageBufferSize))
		b.Reset()
		return b
	},
}

// Get returns a buffer from the pool.
func Get() (buf *bytes.Buffer) {
	return bufferPool.Get().(*bytes.Buffer)
}

// Put returns a buffer to the pool.
// The buffer is reset before it is put back into circulation.
func Put(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}
