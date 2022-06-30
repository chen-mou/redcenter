package redcenter

import (
	"RedisRegister/main/client/redcenter/protool"
	"RedisRegister/main/client/redis"
	"RedisRegister/tool/log"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
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
	auth   func(string) (bool, string)
}

func Open(conf *Configure, wconf *WebConfig) (*Server, error) {
	s := &Server{
		conf:  conf,
		wconf: wconf,
	}
	return s, nil
}

func Error(msg string) []byte {
	return []byte("&code:500-&errmsg=" + msg + "-")
}

func Successful(msg string, data interface{}) []byte {
	begin := "&code:200-&{\"msg\":+\"" + msg + "\""
	end := "}-"
	if data == nil {
		return []byte(begin + end)
	}
	jn, _ := json.Marshal(data)
	b := []byte(begin + ",\"data\":" + string(jn) + end)
	return b
}

func Fail(code string, msg string) []byte {
	return []byte("&code:" + code + "-&{\"msg\":\"" + msg + "\"}-")
}

/*
& 开始符表示数据开始
- 结束符表示数据结束
\ 转义符表示下个字符为普通字符
*/
func (s *Server) handler(conn net.Conn) {
	head := make([]byte, 256)
	conn.Read(head)
	defer conn.Close()
	hd, err := headHandler(head)
	if err != nil {
		conn.Write(Error(err.Error()))
		return
	}
	if ok, msg := s.auth(hd.Auth); !ok {
		conn.Write(Fail("403", msg))
		return
	} else {
		conn.Write(Successful("ok", nil))
	}
}

func headHandler(head []byte) (*protool.Head, error) {
	data := make([]byte, 1)
	start := false
	closed := false
	for i := 0; i < len(head) && !closed; i++ {
		if head[i] == '&' {
			start = true
			continue
		}
		if !start {
			continue
		}
		switch head[i] {
		case '-':
			closed = true
		case '\\':
			data = append(data, head[i])
			i++
		default:
			data = append(data, head[i])
		}
	}
	h := &protool.Head{
		Custom: map[string][]string{},
	}
	atrs := strings.Split(string(data), "\n")
	for _, value := range atrs {
		names := strings.Split(value, ":")
		switch names[1] {
		case "Auth", "auth", "AUTH":
			h.Auth = names[2]
		case "Content-Length", "content-length":
			var err error
			h.Length, err = strconv.Atoi(names[2])
			if err != nil {
				return nil, errors.New("头属性Content-Length有误")
			}
		default:
			h.Custom[names[0]] = strings.Split(names[1], ",")
		}
	}
	return h, nil
}

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
