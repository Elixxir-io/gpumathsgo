///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

//+build linux,gpu

package gpumaths

import (
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"testing"
)

// Helper functions shared by tests are located in gpu_test.go

func initElGamal(batchSize uint32) (*cyclic.Group, *cyclic.Int,
	*cyclic.IntBuffer, *cyclic.IntBuffer) {
	grp := initTestGroup()

	// Set up Keys and public cypher key for operation
	PublicCypherKey := grp.NewInt(1)
	grp.Random(PublicCypherKey)

	// Set the R and Y_R or other Keys (phase/share)
	phaseKeys := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	shareKeys := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initKeys(grp, batchSize, phaseKeys, shareKeys, int64(42))
	return grp, PublicCypherKey, phaseKeys, shareKeys
}

func elgamalCPU(t testing.TB, batchSize uint32, grp *cyclic.Group,
	PublicCypherKey *cyclic.Int, phaseKeys, shareKeys,
	KeysPayload, CypherPayload *cyclic.IntBuffer) {
	for i := uint32(0); i < batchSize; i++ {
		cryptops.ElGamal(grp,
			phaseKeys.Get(i), shareKeys.Get(i),
			PublicCypherKey,
			KeysPayload.Get(i), CypherPayload.Get(i))
	}
}

func elgamalGPU(t testing.TB, streamPool *StreamPool, batchSize uint32,
	grp *cyclic.Group, PublicCypherKey *cyclic.Int, phaseKeys, shareKeys,
	KeysPayload, CypherPayload *cyclic.IntBuffer) {
	err := ElGamalChunk(streamPool, grp, phaseKeys, shareKeys,
		PublicCypherKey, KeysPayload, CypherPayload)
	if err != nil {
		t.Fatal(err)
	}
}

// Runs precomp decrypt test with GPU stream pool and graphs
func TestElGamal(t *testing.T) {
	batchSize := uint32(1024)
	grp, PublicCypherKey, phaseKeys, shareKeys := initElGamal(batchSize)

	// Generate the payload buffers
	KeysPayloadCPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, KeysPayloadCPU, 42)
	CypherPayloadCPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, CypherPayloadCPU, 43)

	// Make a copy for GPU Processing
	KeysPayloadGPU := KeysPayloadCPU.DeepCopy()
	CypherPayloadGPU := CypherPayloadCPU.DeepCopy()

	// Run CPU
	elgamalCPU(t, batchSize, grp, PublicCypherKey, phaseKeys, shareKeys,
		KeysPayloadCPU, CypherPayloadCPU)

	// Run GPU
	streamPool, err := NewStreamPool(2, 65536)
	if err != nil {
		t.Fatal(err)
	}
	elgamalGPU(t, streamPool, batchSize, grp, PublicCypherKey,
		phaseKeys, shareKeys, KeysPayloadGPU, CypherPayloadGPU)

	printLen := len(grp.GetPBytes()) / 2 // # bits / 16 for hex
	for i := uint32(0); i < batchSize; i++ {
		KPGPU := KeysPayloadGPU.Get(i)
		KPCPU := KeysPayloadCPU.Get(i)
		if KPGPU.Cmp(KPGPU) != 0 {
			t.Errorf("KeysPayloadMisMatch on index %d:\n%s\n%s", i,
				KPGPU.TextVerbose(16, printLen),
				KPCPU.TextVerbose(16, printLen))
		}

		CPGPU := CypherPayloadGPU.Get(i)
		CPCPU := CypherPayloadCPU.Get(i)
		if CPGPU.Cmp(CPGPU) != 0 {
			t.Errorf("CypherPayload mismatch on index %d:\n%s\n%s",
				i, CPGPU.TextVerbose(16, printLen),
				CPCPU.TextVerbose(16, printLen))
		}
	}
	streamPool.Destroy()
}

// BenchmarkElGamalCPU provides a baseline with a single-threaded CPU benchmark
func runElGamalCPU(b *testing.B, batchSize uint32) {
	grp, PublicCypherKey, phaseKeys, shareKeys := initElGamal(batchSize)

	// Generate the payload buffers
	KeysPayloadCPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, KeysPayloadCPU, 42)
	CypherPayloadCPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, CypherPayloadCPU, 43)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elgamalCPU(b, batchSize, grp, PublicCypherKey, phaseKeys,
			shareKeys, KeysPayloadCPU, CypherPayloadCPU)
	}
}

// BenchmarkElGamalGPU provides a basic GPU benchmark
func runElGamalGPU(b *testing.B, batchSize uint32) {
	grp, PublicCypherKey, phaseKeys, shareKeys := initElGamal(batchSize)

	// Generate the payload buffers
	KeysPayloadGPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, KeysPayloadGPU, 42)
	CypherPayloadGPU := grp.NewIntBuffer(batchSize, grp.NewInt(1))
	initRandomIntBuffer(grp, batchSize, CypherPayloadGPU, 43)

	streamPool, err := NewStreamPool(2, 65536)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		elgamalGPU(b, streamPool, batchSize, grp, PublicCypherKey,
			phaseKeys, shareKeys, KeysPayloadGPU, CypherPayloadGPU)
	}
	streamPool.Destroy()
}

func BenchmarkElGamalCPU_N(b *testing.B) {
	runElGamalCPU(b, uint32(b.N))
}
func BenchmarkElGamalCPU_1024(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalCPU(b, uint32(1024))
}
func BenchmarkElGamalCPU_8192(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalCPU(b, uint32(1024*8))
}
func BenchmarkElGamalCPU_16384(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalCPU(b, uint32(1024*16))
}
func BenchmarkElGamalCPU_32768(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalCPU(b, uint32(1024*32))
}

func BenchmarkElGamalGPU_N(b *testing.B) {
	runElGamalGPU(b, uint32(b.N))
}
func BenchmarkElGamalGPU_1024(b *testing.B) {
	runElGamalGPU(b, uint32(1024))
}
func BenchmarkElGamalGPU_8192(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalGPU(b, uint32(1024*8))
}
func BenchmarkElGamalGPU_16384(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalGPU(b, uint32(1024*16))
}
func BenchmarkElGamalGPU_32768(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}
	runElGamalGPU(b, uint32(1024*32))
}

// BenchmarkElGamalCUDA4096_256_streams benchmarks the ElGamal and stream functions directly.
func BenchmarkElGamalCUDA4096_256_streams(b *testing.B) {
	const xBitLen = 4096
	const xByteLen = xBitLen / 8
	const yBitLen = 256
	const yByteLen = yBitLen / 8
	g := makeTestGroup4096()
	env := gpumaths4096{}
	// Use two streams with 32k items per kernel launch
	numItems := 32768

	// OK, this shouldn't cause the test to run forever if the stream size is smaller than it should be (like this)
	// In real-world usage, the number of slots passed in should be determined by what the stream supports
	//  (i.e. check stream.MaxSlotsElgamal)
	streamPool, err := NewStreamPool(2, env.streamSizeContaining(numItems, kernelElgamal))
	if err != nil {
		b.Fatal(err)
	}
	// Using prng because the cryptographically secure RNG used by the group is too slow to feed the GPU
	rng := newRng(5)
	b.ResetTimer()
	remainingItems := b.N
	for i := 0; i < b.N; i += numItems {
		// If part of a chunk remains, only upload that part
		remainingItems = b.N - i
		numItemsToUpload := numItems
		if remainingItems < numItems {
			numItemsToUpload = remainingItems
		}
		input := ElGamalInput{
			Slots:           make([]ElGamalInputSlot, numItemsToUpload),
			PublicCypherKey: g.Random(g.NewInt(1)).Bytes(),
			Prime:           g.GetPBytes(),
			G:               g.GetG().Bytes(),
		}
		// Hopefully random number generation doesn't bottleneck things!
		for j := 0; j < numItemsToUpload; j++ {
			// Unfortunately, we can't just generate things using the group, because it's too slow
			key := make([]byte, xByteLen)
			ecrKey := make([]byte, xByteLen)
			cypher := make([]byte, xByteLen)
			privateKey := make([]byte, yByteLen)
			rng.Read(key)
			rng.Read(ecrKey)
			rng.Read(cypher)
			rng.Read(privateKey)
			for !g.BytesInside(key, ecrKey, cypher, privateKey) {
				rng.Read(key)
				rng.Read(ecrKey)
				rng.Read(cypher)
				rng.Read(privateKey)
			}
			input.Slots[j] = ElGamalInputSlot{
				PrivateKey: privateKey,
				Key:        key,
				EcrKey:     ecrKey,
				Cypher:     cypher,
			}
		}
		stream := streamPool.TakeStream()
		resultChan := ElGamal(input, env, stream)
		go func() {
			result := <-resultChan
			streamPool.ReturnStream(stream)
			if result.Err != nil {
				b.Fatal(result.Err)
			}
		}()
	}
	// Empty the pool to make sure results have all been downloaded
	streamPool.TakeStream()
	streamPool.TakeStream()
	b.StopTimer()
	err = streamPool.Destroy()
	if err != nil {
		b.Fatal(err)
	}

	err = resetDevice()
	if err != nil {
		b.Fatal(err)
	}
}
