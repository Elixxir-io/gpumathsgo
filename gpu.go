///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

//+build linux,gpu

package gpumaths

// gpu.go contains helper functions and constants used by
// the gpu implementation. See the exp, elgamal, reveal, or strip _gpu.go
// files for implementations of specific operations.

// When the gpumaths library itself is under development, it should
// use the version of gpumaths that's built in-repository
// (./lib/libpowmosm75.so). golang puts a ./lib entry in the rpath
// itself (as far as I can tell, but it's after everything, so
// to have the ./lib version take priority, there's another entry
// before so the development version takes precedence if both are
// present

/*
#cgo CFLAGS: -I./cgbnBindings/powm -I/opt/xxnetwork/include
#cgo LDFLAGS: -L/opt/xxnetwork/lib -lpowmosm75 -Wl,-rpath,./lib:/opt/xxnetwork/lib
#include <powm_odd_export.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"errors"
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"reflect"
	"unsafe"
)

type gpumathsEnv interface {
	// enqueue calls put, run, and download all together
	enqueue(stream Stream, whichToRun C.enum_kernel, numSlots int) error
	download(stream Stream) error
	run(stream Stream) error
	put(stream Stream, whichToRun C.enum_kernel, numSlots int) error
	getBitLen() int
	getByteLen() int
	getConstantsSize(C.enum_kernel) int
	getOutputSize(C.enum_kernel) int
	getInputSize(C.enum_kernel) int
	getCpuOutputs(stream Stream) unsafe.Pointer
	getCpuInputs(stream Stream, kernel C.enum_kernel) unsafe.Pointer
	maxSlots(memSize int, op C.enum_kernel) int
	streamSizeContaining(numItems int, kernel int) int
}

// TODO These types implement gpumaths? interface
type (
	gpumaths2048 struct{}
	gpumaths3200 struct{}
	gpumaths4096 struct{}
)

func chooseEnv(g *cyclic.Group) gpumathsEnv {
	primeLen := g.GetP().BitLen()
	len2048 := gpumaths2048{}.getBitLen()
	len3200 := gpumaths3200{}.getBitLen()
	len4096 := gpumaths4096{}.getBitLen()
	if primeLen <= len2048 {
		return gpumaths2048{}
	} else if primeLen <= len3200 {
		return gpumaths3200{}
	} else if primeLen <= len4096 {
		return gpumaths4096{}
	} else {
		panic(fmt.Sprintf("Prime %s was too big for any available gpumaths environment", g.GetP().Text(16)))
	}
}

func (gpumaths2048) getCpuOutputs(stream Stream) unsafe.Pointer {
	return C.getCpuOutputs2048(stream.s)
}
func (gpumaths3200) getCpuOutputs(stream Stream) unsafe.Pointer {
	return C.getCpuOutputs3200(stream.s)
}
func (gpumaths4096) getCpuOutputs(stream Stream) unsafe.Pointer {
	return C.getCpuOutputs4096(stream.s)
}

func (gpumaths2048) getCpuInputs(stream Stream, kernel C.enum_kernel) unsafe.Pointer {
	return C.getCpuInputs2048(stream.s, kernel)
}
func (gpumaths3200) getCpuInputs(stream Stream, kernel C.enum_kernel) unsafe.Pointer {
	return C.getCpuInputs3200(stream.s, kernel)
}
func (gpumaths4096) getCpuInputs(stream Stream, kernel C.enum_kernel) unsafe.Pointer {
	return C.getCpuInputs4096(stream.s, kernel)
}

func (gpumaths2048) getBitLen() int {
	return 2048
}
func (gpumaths2048) getByteLen() int {
	return 2048 / 8
}
func (gpumaths3200) getBitLen() int {
	return 3200
}
func (gpumaths3200) getByteLen() int {
	return 3200 / 8
}
func (gpumaths4096) getBitLen() int {
	return 4096
}
func (gpumaths4096) getByteLen() int {
	return 4096 / 8
}

// Create byte slice viewing memory at a certain memory address with a
// certain length
// Here be dragons
func toSlice(pointer unsafe.Pointer, size int) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&reflect.SliceHeader{Data: uintptr(pointer),
			Len: size, Cap: size}))
}

// Load the shared library and return any errors
// Copies a C string into a Go error and frees the C string
// TODO is this slow?
func goError(cString *C.char) error {
	if cString != nil {
		errorStringGo := C.GoString(cString)
		err := errors.New(errorStringGo)
		C.free((unsafe.Pointer)(cString))
		return err
	}
	return nil
}

// Creates streams of a particular size meant to run a particular operation
// TODO This ideally shouldn't need variants
//  Maxslots should exist for each size variant
//  (or just calculate it)
func createStreams(numStreams int, capacity int) ([]Stream, error) {
	streamCreateInfo := C.struct_streamCreateInfo{
		capacity: C.size_t(capacity),
	}

	streams := make([]Stream, 0, numStreams)

	for i := 0; i < numStreams; i++ {
		createStreamResult := C.createStream(streamCreateInfo)
		if createStreamResult.result != nil {
			streams = append(streams, Stream{
				s:       createStreamResult.result,
				memSize: capacity,
			})
		}
		if createStreamResult.error != nil {
			// Try to destroy all created streams to avoid leaking memory
			for j := 0; j < len(streams); j++ {
				C.destroyStream(streams[j].s)
			}
			return nil, goError(createStreamResult.error)
		}
	}

	return streams, nil
}

func destroyStreams(streams []Stream) error {
	for i := 0; i < len(streams); i++ {
		err := C.destroyStream(streams[i].s)
		if err != nil {
			return goError(err)
		}
	}
	return nil
}

// Calculate x**y mod p using CUDA
// Results are put in a byte array for translation back to cyclic ints elsewhere
// Currently, we upload and execute all in the same method

// Upload some items to the next stream
// Returns the stream that the data were uploaded to
// TODO Store the kernel enum for the upload in the stream
//  That way you don't have to pass that info again for run
//  There should be no scenario where the stream gets run for a different kernel than the upload
func (gpumaths2048) enqueue(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.enqueue2048(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}
func (gpumaths3200) enqueue(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.enqueue3200(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}
func (gpumaths4096) enqueue(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.enqueue4096(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}
func (gpumaths2048) put(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.upload2048(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}
func (gpumaths3200) put(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.upload3200(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}
func (gpumaths4096) put(stream Stream, whichToRun C.enum_kernel, numSlots int) error {
	uploadError := C.upload4096(C.uint(numSlots), stream.s, whichToRun)
	if uploadError != nil {
		return goError(uploadError)
	} else {
		return nil
	}
}

// Can you use the C type like this?
// Might need to redefine enumeration in Golang
func (gpumaths2048) run(stream Stream) error {
	return goError(C.run2048(stream.s))
}
func (gpumaths3200) run(stream Stream) error {
	return goError(C.run3200(stream.s))
}
func (gpumaths4096) run(stream Stream) error {
	return goError(C.run4096(stream.s))
}

// Enqueue a download for this stream after execution finishes
// Doesn't actually block for the download
func (gpumaths2048) download(stream Stream) error {
	return goError(C.download2048(stream.s))
}
func (gpumaths3200) download(stream Stream) error {
	return goError(C.download3200(stream.s))
}
func (gpumaths4096) download(stream Stream) error {
	return goError(C.download4096(stream.s))
}

// Four numbers per input
// Returns size in bytes
func (gpumaths2048) getInputSize(kernel C.enum_kernel) int {
	return int(C.getInputSize2048(kernel))
}
func (gpumaths3200) getInputSize(kernel C.enum_kernel) int {
	return int(C.getInputSize3200(kernel))
}
func (gpumaths4096) getInputSize(kernel C.enum_kernel) int {
	return int(C.getInputSize4096(kernel))
}

// Returns size in bytes
func (gpumaths2048) getOutputSize(kernel C.enum_kernel) int {
	return int(C.getOutputSize2048(kernel))
}
func (gpumaths3200) getOutputSize(kernel C.enum_kernel) int {
	return int(C.getOutputSize3200(kernel))
}
func (gpumaths4096) getOutputSize(kernel C.enum_kernel) int {
	return int(C.getOutputSize4096(kernel))
}

// Returns size in bytes
func (gpumaths2048) getConstantsSize(kernel C.enum_kernel) int {
	return int(C.getConstantsSize2048(kernel))
}
func (gpumaths3200) getConstantsSize(kernel C.enum_kernel) int {
	return int(C.getConstantsSize3200(kernel))
}
func (gpumaths4096) getConstantsSize(kernel C.enum_kernel) int {
	return int(C.getConstantsSize4096(kernel))
}

// Helper functions for sizing
// Get the number of slots for an operation
func (g gpumaths2048) maxSlots(memSize int, op C.enum_kernel) int {
	constantsSize := g.getConstantsSize(op)
	slotSize := g.getInputSize(op) + g.getOutputSize(op)
	memForSlots := memSize - constantsSize
	if memForSlots < 0 {
		return 0
	} else {
		return memForSlots / slotSize
	}
}
func (g gpumaths3200) maxSlots(memSize int, op C.enum_kernel) int {
	constantsSize := g.getConstantsSize(op)
	slotSize := g.getInputSize(op) + g.getOutputSize(op)
	memForSlots := memSize - constantsSize
	if memForSlots < 0 {
		return 0
	} else {
		return memForSlots / slotSize
	}
}
func (g gpumaths4096) maxSlots(memSize int, op C.enum_kernel) int {
	constantsSize := g.getConstantsSize(op)
	slotSize := g.getInputSize(op) + g.getOutputSize(op)
	memForSlots := memSize - constantsSize
	if memForSlots < 0 {
		return 0
	} else {
		return memForSlots / slotSize
	}
}

func (g gpumaths2048) streamSizeContaining(numItems int, kernel int) int {
	return g.getInputSize(C.enum_kernel(kernel))*numItems +
		g.getOutputSize(C.enum_kernel(kernel))*numItems +
		g.getConstantsSize(C.enum_kernel(kernel))
}
func (g gpumaths3200) streamSizeContaining(numItems int, kernel int) int {
	return g.getInputSize(C.enum_kernel(kernel))*numItems +
		g.getOutputSize(C.enum_kernel(kernel))*numItems +
		g.getConstantsSize(C.enum_kernel(kernel))
}
func (g gpumaths4096) streamSizeContaining(numItems int, kernel int) int {
	return g.getInputSize(C.enum_kernel(kernel))*numItems +
		g.getOutputSize(C.enum_kernel(kernel))*numItems +
		g.getConstantsSize(C.enum_kernel(kernel))
}

// Wait for this stream's download to finish and return a pointer to the results
// This also checks the CGBN error report (presumably this is where things should be checked, if not now, then in the future, to see whether they're in the group or not. However this may not(?) be doable if everything is in Montgomery space.)
// TODO Copying results to Golang should no longer be the responsibility of this method
//  Instead, this can be done in the exported integration method, and it can be copied from
//  the results buffer directly. The length of the results buffer could be another
//  field of the struct as well, although it would be better to allocate that earlier.
func get(stream Stream) error {
	return goError(C.getResults(stream.s))
}

// get2 queries the stream to see if the event has completed
//func get2(stream Stream) error {
//	return goError(C.getResults2(stream.s))
//}

// Reset the CUDA device
// Hopefully this will allow the CUDA profile to be gotten in the graphical profiler
func resetDevice() error {
	errString := C.resetDevice()
	err := goError(errString)
	return err
}

// TODO better to use an offset or slice the header in different places?
// Puts an integer (in bytes) into a buffer
// Check bounds here? Any safety available?
// Src and dst should be different memory areas. This isn't meant to work meaningfully if the buffers overlap.
// n is the length in bytes of the int in the destination area
// if src is too short, an area of dst will be overwritten with zeroes for safety reasons (right-padded)
func putInt(dst []byte, src []byte, n int) {
	n2 := len(src)
	for i := 0; i < n2; i++ {
		dst[n2-i-1] = src[i]
	}
	for i := n2; i < n; i++ {
		dst[i] = 0
	}
}
