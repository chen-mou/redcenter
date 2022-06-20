package connect

import "net"

const (
	Master   = "master"
	Follower = "follower"
)

type RedisClient struct {
	Conf   *Configure
	conn   net.Conn
	master *RedisClient
}

type Configure struct {
	Host     string
	Port     string
	Password string
	Role     string
	Master   *Configure
}

type ResCmd struct {
	err string
	res string
	len int64
}

func (client *RedisClient) Open(conf *Configure) {
	con, err := net.Dial("tcp", conf.Host+":"+conf.Port)
	if err != nil {
		panic(err)
	}
	client.conn = con
}

func (client *RedisClient) Send(order string) string {
	b := make([]byte, 1024)
	res := ""
	n, err := client.conn.Write([]byte(order + "\r\n"))
	if err != nil {
		panic(err.Error())
	}
	n, err = client.conn.Read(b)
	if err != nil {
		panic(err.Error())
	}
	for n == 1024 {
		res += string(b)
		n, err = client.conn.Read(b)
		if err != nil {
			panic(err.Error())
		}
	}
	res += string(b[:n])
	return res
}
