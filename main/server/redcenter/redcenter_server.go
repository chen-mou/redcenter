package redcenter

import (
	"RedisRegister/main/client/redis"
	"RedisRegister/main/server/redcenter/protool"
	"RedisRegister/tool/log"
	"encoding/json"
	"fmt"
	"net"
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

type DB interface {
	Save(v interface{}) error
	Get(args ...interface{}) (interface{}, error)
}

type Server struct {
	conf   *Configure
	wconf  *WebConfig
	logger log.Logger
	auth   func(string) (bool, string)
	root   *Node
}

func Open(conf *Configure, wconf *WebConfig) (*Server, error) {
	s := &Server{
		conf:   conf,
		wconf:  wconf,
		logger: log.DefaultLogger{},
		auth:   func(string) (bool, string) { return true, "" },
		root: &Node{
			middleWare: make([]func(ctx *Context), 0),
			handler:    map[string]func(ctx *Context){},
			children:   map[string]*Node{},
		},
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
	hd := protool.Header{}
	err := hd.Build(head)
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
	list, err := s.root.find(hd.Path, hd.Method)
	if err != nil {
		conn.Write(Fail("404", err.Error()))
		conn.Close()
		return
	}
	body := make([]byte, hd.Length)
	conn.Read(body)

	bod := protool.Body{}
	bod.Builder(body)

	ctx := &Context{}
	ctx.make(bod, hd, conn)

	list.run(ctx)
}

func (s *Server) Handler(method string, path string, handler func(ctx *Context)) *Server {
	n := s.root
	n.Handler(method, path, handler)
	return s
}

func (s *Server) Get(path string, handler func(ctx *Context)) *Server {
	s.Handler("get", path, handler)
	return s
}

func (s *Server) Post(path string, handler func(ctx *Context)) *Server {
	s.Handler("post", path, handler)
	return s
}

func (s *Server) MiddleWare(handler func(ctx *Context)) *Server {
	s.root.middleWare = append(s.root.middleWare, handler)
	return s
}

func (s *Server) Group(path string) *Node {
	index := strings.LastIndex(path, "/")
	if index == 0 {
		return s.Group(path[1:])
	}
	if index != -1 {
		n, ok := s.root.children[path[:index]]
		if ok {
			return n.Group(path[index+1:])
		}
		n = s.Group(path[:index])
		s.root.children[path[:index]] = n
		path = path[index+1:]
		return n.Group(path)
	} else {
		n, ok := s.root.children[path]
		if ok {
			return n
		}
		n = &Node{
			name:       path,
			typ:        "group",
			middleWare: make([]func(ctx *Context), 0),
			handler:    map[string]func(ctx *Context){},
			children:   map[string]*Node{},
		}
		s.root.children[path] = n
		return n
	}
}

func (s *Server) Run() error {
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
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					msg := err.(error).Error()
					s.logger.Error(msg)
					fmt.Println(msg)
				}
			}()
			s.handler(conn)
		}()
	}
}
