package main

import (
	"RedisRegister/main/client/redcenter/protool"
	"RedisRegister/main/client/redis"
	"encoding/json"
	"flag"
	"fmt"
)

func main() {
	var port int
	flag.IntVar(&port, "p", 7888, "监听的端口")
	conn := redis.Client{
		Conf: &redis.Configure{
			Host: "localhost",
			Port: "6379",
		},
	}
	//listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	//if err != nil {
	//	panic(err)
	//}
	//c, err := listen.Accept()
	//c.
	conn.Open()

	conn.HSet("Head", protool.Head{
		Auth: "123456789",
		Custom: map[string][]string{
			"Content-Type": {"application/json", "application/xml"},
			"Method":       {"Post", "Get", "Options"},
		},
	})
	fmt.Println(conn.Execute("HGET Head.Custom Content-Type").Result())
	val := protool.Head{}
	conn.HGet("Head", &val)
	b, _ := json.Marshal(val)
	fmt.Println(string(b))
}
