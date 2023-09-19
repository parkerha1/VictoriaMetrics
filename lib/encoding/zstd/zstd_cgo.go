//go:build cgo
// +build cgo

package zstd

import (
	"fmt"

	"github.com/intel/qatgo/qatzip"
	"github.com/valyala/gozstd"
)

// Decompress appends decompressed src to dst and returns the result.
func Decompress(dst, src []byte) ([]byte, error) {
	return gozstd.Decompress(dst, src)
}

// CompressLevel appends compressed src to dst and returns the result.
//
// The given compressionLevel is used for the compression.
func CompressLevel(dst, src []byte, compressionLevel int) []byte {
	if cap(dst) == 0 {
		dst = append(dst, make([]byte, len(src))...)
	}

	binding, err := qatzip.NewQzBinding()
	if err != nil {
		fmt.Printf("error: %v\n", err)
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
	dst = dst[:cap(dst)]
	binding.SetLast(true)

	for {
		_, out, err := binding.Compress(src, dst)
		if err != nil {
			if err == qatzip.ErrBuffer {
				dst = append(dst, make([]byte, len(dst))...)
				//log.Panicln("Panic: resize src: ", len(src), " dst: ", len(dst2))
				continue
			} else {
				fmt.Printf("COMPRESS ERROR: %v\n", err)
				fmt.Printf("Location of first src entry: %p\n", &src[0])
				fmt.Printf("Location of first dst entry: %p\n", &dst[0])

				panic("Error")
			}
		}
		dst = dst[:out]
		break
	}
	//binding.Close() //Currently this Close() will cause VM to crash, still mostly functions without it 
	return dst
	//return gozstd.CompressLevel(dst, src, compressionLevel)
}
