package connect

import (
	"context"
	"errors"
	"time"
)

type Conn interface {
	Close() error
	Open() error
}

type Pool struct {
	core chan *Conn
	max  chan *Conn
	wait chan *Conn
}

func (p *Pool) Build(coreNum, maxNum, waitLen int) *Pool {
	p.core = make(chan *Conn, coreNum)
	p.max = make(chan *Conn, maxNum)
	p.wait = make(chan *Conn, waitLen)
	return p
}

func (p Pool) Submit(tim int64, f func(*Conn) interface{}, reject func()) (<-chan interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond*time.Duration(tim))
	ch := make(chan *Conn, 1)
	res := make(chan interface{}, 1)
	go func() {
		c, ok := <-p.core
		if !ok {
			c, ok = <-p.wait
			if !ok {
				c = <-p.max
			}
		}
		cancel()
		ch <- c
	}()
	select {
	case <-ctx.Done():
		reject()
		return nil, errors.New("timeout")
	case conn := <-ch:
		go func() {
			res <- f(conn)
		}()
		return res, nil
	}
}
