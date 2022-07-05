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
	if !ok {
		return errors.New("这个连接不属于这个容器")
	}
	if index >= c.windex {
		return errors.New("当前连接已经归还")
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
		core:        build(coreNum),
		wait:        make(chan *wait, waitLen),
		builder:     builder,
		max:         nil,
		maxN:        maxNum,
		maxLiveTime: time.Minute * 20,
	}
	go func(p *Pool) {
		for {
			select {
			case w := <-p.wait:
				HandWait(p.core, w, p.builder)
			}
		}
	}(p)
	return p
}

func openMax(p *Pool) {
	ctx, _ := context.WithTimeout(context.Background(), p.maxLiveTime)
	for {
		select {
		case <-ctx.Done():
			p.max.Close()
			p.max = nil
			return
		case w := <-p.wait:
			ctx, _ = context.WithTimeout(context.Background(), p.maxLiveTime)
			HandWait(p.max, w, p.builder)
		}
	}
}

func (p *Pool) Submit(tim int64, f func(Conn) interface{}, reject func()) (<-chan interface{}, error) {
	type chElem struct {
		c *contain

		con Conn
	}
	ch := make(chan *chElem, 1)
	res := make(chan interface{}, 1)

	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(tim))

	for {
		select {
		case <-ctx.Done():
			//超时处理
			go func() {
				elem, ok := <-ch
				if !ok || elem == nil {
					return
				}
				elem.c.backIdleConn(elem.con)
			}()
			if reject != nil {
				reject()
			}
			return nil, errors.New("timeout")
		case elem := <-ch:
			//获取连接后处理连接
			if elem == nil {
				return res, nil
			}
			go func() {
				res <- f(elem.con)
				elem.c.backIdleConn(elem.con)
				close(res)
			}()
			return res, nil
		default:
			//尝试获取连接
			f1 := func(c1 *contain, conn Conn) {
				elem := &chElem{
					c1, conn,
				}
				ch <- elem
				close(ch)
			}
			conn, err := p.core.getIdleConn(p.builder)
			if err != nil {
				select {
				case p.wait <- &wait{res, f}:
					ch <- nil
				default:
					if p.max == nil {
						//开启额外的连接处理队列
						p.Lock()
						if p.max == nil {
							p.max = build(p.maxN)
							go openMax(p)
						}
						p.Unlock()
					}
					conn, err = p.max.getIdleConn(p.builder)
					if err == nil {
						f1(p.max, conn)
					}
				}
			} else {
				f1(p.core, conn)
			}
		}
	}
}
