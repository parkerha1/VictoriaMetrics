//go:build cgo
// +build cgo

package zstd

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/intel/qatgo/qatzip"
	"github.com/valyala/gozstd"
)

const (
	envWorkers     = "QAT_ZSTD_WORKERS"
	DefaultWorkers = 8
)

var (
	once     sync.Once
	sessions []session
)

type session struct {
	binding *qatzip.QzBinding
	m       sync.Mutex
}

func acquireBinding() int {
	for {
		for a := range sessions {
			if sessions[a].m.TryLock() {
				return a
			}
		}
	}
}

// Decompress appends decompressed src to dst and returns the result.
func Decompress(dst, src []byte) ([]byte, error) {
	return gozstd.Decompress(dst, src)
}

func initBinding() {
	var err error
	var workerThreads int
	workerThreadsStr := os.Getenv(envWorkers)
	if workerThreadsStr != "" {
		if workerThreads, err = strconv.Atoi(workerThreadsStr); err != nil {
			panic("Invalid " + envWorkers + " value")
		}
	} else {
		workerThreads = DefaultWorkers
	}
	sessions = make([]session, workerThreads)
	for x := 0; x < len(sessions); x++ {
		binding, err := qatzip.NewQzBinding()
		if err != nil {
			fmt.Printf("BINDING ERROR: %v\n", err)
			panic("Error")
		}
		err = binding.Apply(qatzip.AlgorithmOption(qatzip.ZSTD))
		if err != nil {
			fmt.Printf("OPTIONS ERROR: %v\n", err)
			panic("Error")
		}
		err = binding.StartSession()
		if err != nil {
			fmt.Printf("SESSION ERROR: %v\n", err)
			panic("Error")
		}
		binding.SetLast(true)
		sessions[x].binding = binding
	}
}

// CompressLevel appends compressed src to dst and returns the result.
//
// The given compressionLevel is used for the compression.
func CompressLevel(dst, src []byte, compressionLevel int) []byte {
	once.Do(initBinding)
	dstLen := len(dst)
	if cap(dst)-len(dst) == 0 {
		dst = append(dst, make([]byte, len(src))...)
	}

	dst = dst[:cap(dst)]

	for {
		sessionID := acquireBinding()
		binding := sessions[sessionID].binding
		binding.Apply(qatzip.CompressionLevelOption(compressionLevel))
		_, out, err := binding.Compress(src, dst[dstLen:])
		sessions[sessionID].m.Unlock()
		if err != nil {
			if err == qatzip.ErrBuffer {
				dst = append(dst, make([]byte, cap(dst)+len(src))...)
				continue
			} else {
				fmt.Printf("COMPRESS ERROR: %v\n", err)
				panic("Error")
			}
		}
		dst = dst[:out+dstLen]
		break
	}
	return dst
	//return gozstd.CompressLevel(dst, src, compressionLevel)
}
