package chunk

import (
	"bytes"
	"context"
	"time"

	"github.com/pachyderm/pachyderm/v2/src/internal/miscutil"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachhash"
	"github.com/pachyderm/pachyderm/v2/src/internal/pachsql"
	"github.com/pachyderm/pachyderm/v2/src/internal/storage/kv"
	"github.com/pachyderm/pachyderm/v2/src/internal/storage/track"
	"github.com/pachyderm/pachyderm/v2/src/internal/stream"
)

const (
	// TrackerPrefix is the prefix used when creating tracker objects for chunks
	TrackerPrefix        = "chunk/"
	prefix               = "chunk"
	defaultChunkTTL      = 30 * time.Minute
	DefaultPrefetchLimit = 10
)

// Storage is an abstraction for interfacing with chunk storage.
// A storage instance:
// - Provides methods for uploading data to chunks and downloading data from chunks.
// - Manages tracker state to keep chunks alive while uploading.
// - Manages an internal chunk cache and work deduplicator (parallel downloads of the same chunk will be deduplicated).
type Storage struct {
	db            *pachsql.DB
	tracker       track.Tracker
	store         kv.Store
	memCache      kv.GetPut
	deduper       *miscutil.WorkDeduper[pachhash.Output]
	prefetchLimit int

	createOpts CreateOptions
}

// NewStorage creates a new Storage.
func NewStorage(objStor kv.Store, memCache kv.GetPut, db *pachsql.DB, tracker track.Tracker, opts ...StorageOption) *Storage {
	s := &Storage{
		db:            db,
		store:         objStor,
		tracker:       tracker,
		memCache:      memCache,
		deduper:       &miscutil.WorkDeduper[pachhash.Output]{},
		prefetchLimit: DefaultPrefetchLimit,
		createOpts: CreateOptions{
			Compression: CompressionAlgo_GZIP_BEST_SPEED,
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewReader creates a new Reader.
func (s *Storage) NewReader(ctx context.Context, dataRefs []*DataRef, opts ...ReaderOption) *Reader {
	client := NewClient(s.store, s.db, s.tracker, nil)
	return newReader(ctx, client, s.memCache, s.deduper, s.prefetchLimit, dataRefs, opts...)
}

func (s *Storage) NewDataReader(ctx context.Context, dataRef *DataRef) *DataReader {
	client := NewClient(s.store, s.db, s.tracker, nil)
	return newDataReader(ctx, client, s.memCache, s.deduper, dataRef, 0)
}

func (s *Storage) PrefetchData(ctx context.Context, dataRef *DataRef) error {
	return s.NewDataReader(ctx, dataRef).fetchData()
}

// List lists all of the chunks in object storage.
func (s *Storage) List(ctx context.Context, cb func(id ID) error) error {
	it := s.store.NewKeyIterator(kv.Span{})
	return stream.ForEach(ctx, it, func(key []byte) error {
		var id ID
		copy(id[:], key)
		return cb(id)
	})
}

// NewDeleter creates a deleter for use with a tracker.GC
func (s *Storage) NewDeleter() track.Deleter {
	return &deleter{}
}

// Check runs an integrity check on the objects in object storage.
// It will check objects for chunks with IDs in the range [first, last)
// As a special case: if len(end) == 0 then it is ignored.
func (s *Storage) Check(ctx context.Context, begin, end []byte, readChunks bool) (int, error) {
	c := NewClient(s.store, s.db, s.tracker, nil).(*trackedClient)
	first := append([]byte{}, begin...)
	var count int
	for {
		n, last, err := c.CheckEntries(ctx, first, 100, readChunks)
		count += n
		if err != nil {
			return count, err
		}
		if last == nil {
			break
		}
		if len(end) > 0 && bytes.Compare(last, end) > 0 {
			break
		}
		first = keyAfter(last)
	}
	return count, nil
}

// keyAfter returns a byte slice ordered immediately after x lexicographically
// the motivating use case is iteration.
func keyAfter(x []byte) []byte {
	y := append([]byte{}, x...)
	y = append(y, 0x00)
	return y
}
