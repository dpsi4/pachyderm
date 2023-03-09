package kv

import (
	"context"
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/pachyderm/pachyderm/v2/src/internal/log"
	"github.com/pachyderm/pachyderm/v2/src/internal/stream"
	"go.uber.org/zap"
)

var _ Store = &LRUCache{}

type LRUCache struct {
	slow, fast Store
	mu         sync.RWMutex
	cache      simplelru.LRU

	ctx context.Context
}

func NewLRUCache(slow, fast Store, size int) *LRUCache {
	c := &LRUCache{
		slow: slow,
		fast: fast,
	}
	cache, err := simplelru.NewLRU(size, c.onEvict)
	if err != nil {
		panic(err)
	}
	c.cache = *cache
	return c
}

func (c *LRUCache) Get(ctx context.Context, key []byte, buf []byte) (int, error) {
	// note that we don't need a lock here because stores are thread-safe and Put/Deletes are atomic.
	n, err := c.fast.Get(ctx, key, buf)
	if err == nil {
		// cache hit
		return n, nil
	}
	// cache miss
	n, err = c.slow.Get(ctx, key, buf)
	if err != nil {
		return 0, err
	}
	if err := c.putFast(ctx, key, buf[:n]); err != nil {
		log.Error(ctx, "kv.LRUCache could not put in fast layer", zap.Any("err", err))
	}
	return n, nil
}

func (c *LRUCache) Put(ctx context.Context, key, value []byte) error {
	return c.slow.Put(ctx, key, value)
}

func (c *LRUCache) Exists(ctx context.Context, key []byte) (bool, error) {
	if exists, err := func() (bool, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.fast.Exists(ctx, key)
	}(); err == nil {
		// cache hit
		return exists, err
	}
	return c.slow.Exists(ctx, key)
}

func (c *LRUCache) Delete(ctx context.Context, key []byte) error {
	if err := c.slow.Delete(ctx, key); err != nil {
		return err
	}
	if err := c.fast.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

func (c *LRUCache) NewKeyIterator(span Span) stream.Iterator[[]byte] {
	return c.slow.NewKeyIterator(span)
}

func (c *LRUCache) putFast(ctx context.Context, key, value []byte) error {
	key = append([]byte{}, key...)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx = ctx
	defer func() { c.ctx = nil }()
	if err := c.fast.Put(ctx, key, value); err != nil {
		return err
	}
	c.cache.Add(key, nil)
	return nil
}

func (c *LRUCache) onEvict(key, value interface{}) {
	k := key.([]byte)
	if err := c.Delete(c.ctx, k); err != nil {
		log.Error(c.ctx, "deleting from cache", zap.Any("err", err))
	}
}
