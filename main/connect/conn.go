package connect

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Conn interface {
	Close() error
	Open() error
}

type ConnBuilder interface {
	Builder() Conn
}

type ContainFullError struct{}

func (ContainFullError) Error() string {
	return "容器满了"
}

type contain struct {
	sync.Mutex

	closing bool

	connIndex map[Conn]int

	arr []Conn

	//已创建的连接个数
	le int
	//最大创建连接个数
	max int
	//最后一个忙碌的连接个数
	windex int
}

func build(max int) *contain {
	return &contain{
		connIndex: make(map[Conn]int),
		arr:       make([]Conn, max),
		le:        0,
		max:       max,
		windex:    0,
	}
}

func (c *contain) getIdleConn(connb ConnBuilder) (conn Conn, err error) {
	c.Lock()
	defer c.Unlock()
	if c.closing {
		return nil, errors.New("容器已经关闭")
	}
	if c.le < c.max {
		c.arr[c.le] = c.arr[c.windex]
		c.arr[c.windex] = connb.Builder()
		c.le++
		conn = c.arr[c.windex]
		c.connIndex[conn] = c.windex
		c.windex++
		err = nil
		return
	} else if c.windex < c.max {
		conn = c.arr[c.windex]
		c.connIndex[conn] = c.windex
		err = nil
		c.windex++
		return
	}
	conn = nil
	err = ContainFullError{}
	return
}

func (c *contain) backIdleConn(conn Conn) error {
	c.Lock()
	defer c.Unlock()
	if c.closing {
		return errors.New("容器已经关闭")
	}
	index, ok := c.connIndex[conn]
	if index >= c.windex {
		return errors.New("当前连接已经归还")
	}
	if !ok {
		return errors.New("这个连接不属于这个容器")
	}
	temp := c.arr[c.windex-1]
	c.connIndex[temp] = index
	c.connIndex[conn] = c.windex - 1
	c.arr[c.windex-1] = conn
	c.arr[index] = temp
	c.windex--
	return nil
}

func (c *contain) Close() {
	c.Lock()
	c.closing = true
	c.Unlock()
	for _, v := range c.arr {
		v.Close()
	}
}

type Pool struct {
	builder ConnBuilder

	core *contain

	max *contain

	wait chan *wait

	maxLiveTime time.Duration

	maxN int

	sync.Mutex

	closed chan struct{}
}

type wait struct {
	ch chan interface{}

	f func(Conn) interface{}
}

func HandWait(contain *contain, w *wait, builder ConnBuilder) {
	conn, err := contain.getIdleConn(builder)
	for err != nil {
		conn, err = contain.getIdleConn(builder)
	}
	w.ch <- w.f(conn)
	contain.backIdleConn(conn)
}

func Build(coreNum, maxNum, waitLen int, builder ConnBuilder) *Pool {
	p := &Pool{
		core:    build(coreNum),
		wait:    make(chan *wait, waitLen),
		builder: builder,
		max:     nil,
		maxN:    maxNum,
	}
	go func() {
		for {
			select {
			case w := <-p.wait:
				HandWait(p.core, w, p.builder)
			}
		}
	}()
	return p
}

func (p *Pool) Submit(tim int64, f func(Conn) interface{}, reject func()) (<-chan interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Microsecond*time.Duration(tim))
	ch := make(chan Conn, 1)
	res := make(chan interface{}, 1)
	var c *contain
	go func() {
		for {
			conn, err := p.core.getIdleConn(p.builder)
			if err != nil {
				select {
				case p.wait <- &wait{res, f}:
					ch <- nil
					cancel()
					return
				default:
					conn, err = p.max.getIdleConn(p.builder)
					if err == nil {
						//开启额外的连接处理队列
						p.Lock()
						if p.max == nil {
							p.max = build(p.maxN)
							go func() {
								t := int64(0)
								for {
									select {
									case <-p.closed:
										p.max = nil
										return
									case w := <-p.wait:
										t = 0
										HandWait(p.max, w, p.builder)
									default:
										t++
									}
									if t == int64(p.maxLiveTime) {
										p.closed <- struct{}{}
									}
								}
							}()
						}
						p.Unlock()
						c = p.max
						ch <- conn
						cancel()
						return
					}
				}
			} else {
				c = p.core
				ch <- conn
				break
			}
		}
		cancel()
	}()
	select {
	case <-ctx.Done():
		reject()
		return nil, errors.New("timeout")
	case conn := <-ch:
		if conn == nil {
			return res, nil
		}
		go func() {
			res <- f(conn)
			c.backIdleConn(conn)
		}()
		return res, nil
	}
}
