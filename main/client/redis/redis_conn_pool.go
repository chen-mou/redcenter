package redis

import "RedisRegister/main/connect"

type RPool connect.Pool

type redisConnBuilder struct {
	Conf *Configure
}

func (r redisConnBuilder) Builder() connect.Conn {
	c := &Client{
		Conf: r.Conf,
	}
	c.Open()
	return c
}

var Pool *RPool

func (p *RPool) Dail(conf *Configure) error {
	Pool = (*RPool)(connect.Build(4, 8, 10, redisConnBuilder{
		Conf: conf,
	}))
	return nil
}

func (p *RPool) Submit(tim int64, f func(conn *Client) interface{}, reject func()) (<-chan interface{}, error) {
	pool := connect.Pool(*p)
	fun := func(conn1 connect.Conn) interface{} {
		return f(conn1.(*Client))
	}
	return pool.Submit(tim, fun, reject)
}
