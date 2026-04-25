//go:build !linux || !cgo

package corehost

import "fmt"

func NewDlopenTransport(path string) (ByteTransport, error) {
	if path == "" {
		return nil, fmt.Errorf("missing shared library path")
	}
	return nil, fmt.Errorf("shared library scene host requires linux+cgo")
}
