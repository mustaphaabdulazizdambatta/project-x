//go:build !linux

package main

func makeRawTerm(fd int) (interface{}, error) {
	return nil, nil
}

func restoreRawTerm(fd int, state interface{}) {}
