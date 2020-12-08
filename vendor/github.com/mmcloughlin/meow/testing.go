package meow

import "math/rand"

// checksumFunc is a method of computing a Meow checksum.
type checksumFunc func(uint64, []byte) []byte

// checksumSlice adapts Checksum to return a slice instead of an array.
func checksumSlice(seed uint64, data []byte) []byte {
	cksum := Checksum(seed, data)
	return cksum[:]
}

// checksumHash implements Checksum with the hash.Hash interface. Intended to facilitate comparison between the two.
func checksumHash(seed uint64, data []byte) []byte {
	h := New(seed)
	h.Write(data)
	return h.Sum(nil)
}

// checksumHashWithReset is intended to confirm hash.Hash Reset() behavior.
// Hashes some random data, resets and then computes the desired hash.
func checksumHashWithReset(seed uint64, data []byte) []byte {
	n := rand.Intn(8 << 10)
	r := make([]byte, n)
	rand.Read(r)

	h := New(seed)
	h.Write(r)
	h.Reset()
	h.Write(data)
	return h.Sum(nil)
}

// checksumRandomBatchedHash implements Checksum by writing random amounts to a hash.Hash.
func checksumRandomBatchedHash(seed uint64, data []byte) []byte {
	h := New(seed)
	for len(data) > 0 {
		n := rand.Intn(len(data) + 1)
		h.Write(data[:n])
		data = data[n:]
	}
	return h.Sum(nil)
}

// checksumHashWithIntermediateSum computes the checksum and calls Sum()
// inbetween. Intended to confirm that Sum() does not change hash state.
func checksumHashWithIntermediateSum(seed uint64, data []byte) []byte {
	h := New(seed)
	half := len(data) / 2
	h.Write(data[:half])
	_ = h.Sum(nil)
	h.Write(data[half:])
	return h.Sum(nil)
}

// checksumPureGo computes the checksum with the fallback Go implementation.
func checksumPureGo(seed uint64, data []byte) []byte {
	cksum := make([]byte, Size)
	checksumgo(seed, cksum, data)
	return cksum
}
