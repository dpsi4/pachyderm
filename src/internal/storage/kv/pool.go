package kv

import (
	"context"
	"sync"
)

// TODO: remove this, let the other storage systems manage their own buffer pools
var DefaultPool = NewPool(20 * 1 << 20)

type Pool struct {
	maxSize int
	pool    sync.Pool
}

func NewPool(maxSize int) *Pool {
	return &Pool{
		maxSize: maxSize,
		pool: sync.Pool{
			New: func() any {
				return make([]byte, maxSize)
			},
		},
	}
}

func (p *Pool) GetF(ctx context.Context, s Getter, key []byte, cb ValueCallback) error {
	buf := p.acquire()
	defer p.release(buf)
	n, err := s.Get(ctx, key, buf)
	if err != nil {
		return err
	}
	return cb(buf[:n])
}

func (p *Pool) acquire() []byte {
	return p.pool.Get().([]byte)
}

func (p *Pool) release(x []byte) {
	p.pool.Put(x)
}
