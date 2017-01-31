// Benchmarks for the hashtree library. How long operations take can depend
// heavily on how much rehashing they do. All times are measured on msteffen's
// Dell XPS laptop with 8 cores and 16GB of RAM.
//
// TODO(msteffen): repeat experiments on GCP, in case they need to be
// reproduced later (though times shouldn't vary all that much, modulo a few
// milliseconds. The important thing is to limit # hashes for large trees,
// which can cause these tests to go from <5s to >1h if done wrong).

package hashtree

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/golang/protobuf/proto"
)

// BenchmarkPutFile tests the amount of time it takes to PutFile 'cnt' files
// into a HashTree.
//
// Because "/foo" and "/" in 'h' must be rehashed after each PutFile, and the
// amount of time it takes to do the rehashing is proportional to the number of
// files already in 'h', this is an O(n^2) operation with respect to 'cnt'.
// Because of this, BenchmarkPutFile can be very slow for large 'cnt', often
// larger then BenchmarkMerge for the same 'cnt'. Be sure to set -timeout 3h for
// 'cnt' == 100k
//  cnt |  time (s)
// -----+-------------
// 1k   |  0.000 s/op
// 10k  | 39.134 s/op
// 100k | - (probably >1h)
func BenchmarkPutFile(b *testing.B) {
	// Add 'cnt' files
	cnt := int(1e4)
	r := rand.New(rand.NewSource(0))
	h := &HashTree{}
	for i := 0; i < cnt; i++ {
		h.PutFile(fmt.Sprintf("/foo/shard-%05d", i),
			br(fmt.Sprintf(`block{hash:"%x"}`, r.Uint32())))
	}
}

// BenchmarkMerge times how long it takes to merge 'cnt' trees, each of which
// has a single small file, into one central hash tree. This is similar to what
// happens at the completion of a job. Because all re-hashing is saved until the
// end, this is O(n) with respect to 'cnt', making it much faster than calling
// PutFile 'cnt' times.
// Benchmarked times at rev. 3ecd3d7520b75b0650f69b3cf4d4ea44908255f8
//  cnt |  time (s)
// -----+-------------
// 1k   | 0.004 s/op
// 10k  | 0.078 s/op
// 100k | 2.732 s/op
func BenchmarkMerge(b *testing.B) {
	// Merge 'cnt' trees, each with 1 file (simulating a job)
	cnt := int(1e5)
	trees := make([]Interface, cnt)
	r := rand.New(rand.NewSource(0))
	for i := 0; i < cnt; i++ {
		trees[i] = new(HashTree)
		trees[i].PutFile(fmt.Sprintf("/foo/shard-%05d", i),
			br(fmt.Sprintf(`block{hash:"%x"}`, r.Uint32())))
	}

	h := HashTree{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Merge(trees)
		h = HashTree{}
	}
}

// BenchmarkClone is idential to BenchmarkDelete, except that it doesn't
// actually call DeleteFile. The idea is to provide a baseline for how long it
// takes to clone a HashTree with 'cnt' elements, so that that number can be
// subtracted from BenchmarkDelete (since that operation is necessarily part of
// the benchmark)
//
// Benchmarked times at rev. 3ecd3d7520b75b0650f69b3cf4d4ea44908255f8
//  cnt |  time (s)
// -----+-------------
// 1k   | 0.003 s/op
// 10k  | 0.042 s/op
// 100k | 0.484 s/op
func BenchmarkClone(b *testing.B) {
	// Create a tree with 'cnt' files
	cnt := int(1e5)
	r := rand.New(rand.NewSource(0))
	srcTs := make([]Interface, cnt)
	for i := 0; i < cnt; i++ {
		srcTs[i] = &HashTree{}
		srcTs[i].PutFile(fmt.Sprintf("/foo/shard-%05d", i),
			br(fmt.Sprintf(`block{hash:"%x"}`, r.Uint32())))
	}
	h := HashTree{}
	h.Merge(srcTs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Clone(&h)
	}
}

// Benchmarked times at rev. 3ecd3d7520b75b0650f69b3cf4d4ea44908255f8
//  cnt |  time (s)
// -----+-------------
// 1k   | 0.004 s/op
// 10k  | 0.039 s/op
// 100k | 0.476 s/op
func BenchmarkDelete(b *testing.B) {
	// Create a tree with 'cnt' files
	cnt := int(1e5)
	r := rand.New(rand.NewSource(0))
	srcTs := make([]Interface, cnt)
	for i := 0; i < cnt; i++ {
		srcTs[i] = &HashTree{}
		srcTs[i].PutFile(fmt.Sprintf("/foo/shard-%05d", i),
			br(fmt.Sprintf(`block{hash:"%x"}`, r.Uint32())))
	}
	h := HashTree{}
	h.Merge(srcTs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h2 := proto.Clone(&h).(*HashTree)
		h2.DeleteFile("/foo")
	}
}
