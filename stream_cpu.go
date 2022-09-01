////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//+build !linux !gpu

package gpumaths

import "errors"

// Stub out all exported symbols with reduced functionality
type Stream struct{}

func (s *Stream) GetMaxSlotsExp() int {
	return 0
}

func (s *Stream) GetMaxSlotsElGamal() int {
	return 0
}

type StreamPool struct{}

func NewStreamPool(numStreams int, memSize int) (*StreamPool, error) {
	return nil, errors.New("gpumaths stubbed build doesn't support CUDA stream pool")
}

func (sm *StreamPool) TakeStream() Stream {
	return Stream{}
}

func (sm *StreamPool) ReturnStream(s Stream) {}

func (sm *StreamPool) Destroy() error {
	return errors.New("gpumaths stubbed build doesn't support CUDA stream pool")
}

func MaxSlots(memSize int, op int) int {
	return 0
}

func streamSizeContaining(numItems int, kernel int) int {
	return 0
}
