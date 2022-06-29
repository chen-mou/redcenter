package redcenter

import (
	"RedisRegister/main/client/redis"
	"RedisRegister/tool/log"
	"net"
)

type Configure struct {
	Host      string
	Port      string
	RedisConf *redis.Configure
}

type WebConfig struct {
	Name     string
	ServerId int
	Index    string
	Verify   func(string) bool
}

type Server struct {
	conf   *Configure
	wconf  *WebConfig
	logger log.Logger
}

func Open(conf *Configure, wconf *WebConfig) (*Server, error) {
	s := &Server{
		conf:  conf,
		wconf: wconf,
	}
	return s, nil
}

func handler(conn net.Conn) {}

func (s Server) Run() error {
	s.logger.Info("准备连接redis")

	err := redis.Pool.Dail(s.conf.RedisConf)
	if err != nil {
		panic(err)
	}

	s.logger.Info("redis连接成功")

	s.logger.Info("准备监听端口:" + s.conf.Port)
	a, err := net.Listen("tcp", s.conf.Host+":"+s.conf.Port)
	for {
		conn, err := a.Accept()
		if err != nil {
			s.logger.Error(err.Error())
			continue
		}
		go handler(conn)
	}
}
