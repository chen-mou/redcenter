package redis

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
)

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

func (client *RedisClient) Open(conf *Configure) {
	con, err := net.Dial("tcp", conf.Host+":"+conf.Port)
	if err != nil {
		panic(err)
	}
	client.conn = con
}

func (client *RedisClient) Set(key string, value interface{}) *ResCmd {
	switch value.(type) {
	case struct{}, map[string]string, map[string]interface{}:
		b, _ := json.Marshal(value)
		return client.send(fmt.Sprintf("set %s %s", key, "\""+strings.Replace(string(b), "\"", "\\\"", -1)+"\""))
	case string:
		return client.send(fmt.Sprintf("set %s %s", key, value))
	case bool:
		return client.send(fmt.Sprintf("set %s %t", key, value))
	case int8, int, int64, int32, uint, uint8, uint32, uint64:
		return client.send(fmt.Sprintf("set %s %d", key, value))
	case float32, float64:
		return client.send(fmt.Sprintf("set %s %f", key, value))
	default:
		panic("类型:" + reflect.TypeOf(value).String() + "不支持")
	}
}

func (client *RedisClient) Get(key string) *ResCmd {
	return client.send(fmt.Sprintf("get %s", key))
}

func (client *RedisClient) HSet(key string, value interface{}) error {
	var handlerStruct func(prefix string, value interface{})
	lua := "redis.call('hset', KEYS[1], KEYS[2], ARGV[1]);"
	keys := make([]string, 2)
	argv := make([]string, 1)
	script := "eval \""
	handlerStruct = func(prefix string, value interface{}) {
		v := reflect.ValueOf(value)
		t := reflect.TypeOf(value)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			t = t.Elem()
		}
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			name := t.Field(i).Name
			switch field.Kind().String() {
			case "struct":
				handlerStruct(prefix+"."+name, field)
			case "map":
				ks := field.MapKeys()
				for _, k := range ks {
					keys = append(keys, prefix, toString(k))
					argv = append(argv, toString(field.MapIndex(k)))
					script += lua
				}
			default:
				keys = append(keys, prefix, name)
				argv = append(argv, toString(field))
				script += lua
			}
		}
	}
	handlerStruct(key, value)
	script = "" + script + "return ok;\" " + strconv.Itoa(len(keys))
	for _, i := range keys {
		script += i
	}
	for _, i := range argv {
		script += i
	}
	client.send("eval" + script)
	return nil
}

func (client *RedisClient) HGetField(key, field string) *ResCmd {

}

func (client *RedisClient) HSetField(key, field string, v interface{}) *ResCmd {

}

func (client *RedisClient) HGet(key string, v interface{}) {

}

func (client *RedisClient) send(order string) *ResCmd {
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
	//fmt.Println(res)
	return handlerResult(res)
}

func toString(val reflect.Value) string {
	kind := val.Kind()
	if kind < 12 && kind > 1 {
		return strconv.FormatInt(val.Int(), 10)
	}
	if kind == 13 {
		return strconv.FormatFloat(val.Float(), 'f', 10, 32)
	} else if kind == 14 {
		return strconv.FormatFloat(val.Float(), 'f', 10, 64)
	}
	if kind == 1 {
		return strconv.FormatBool(val.Bool())
	}
	if kind == 24 {
		return val.String()
	}
	return ""
}

func handlerResult(res string) *ResCmd {
	re := &ResCmd{}
	switch res[0] {
	case '-':
		re.err = res[1:]
		re.len = 0
	case '+':
		re.res = res[1:]
	case '$':
		l := 0
		i := 1
		for ; res[i] != '\r'; i++ {
			l = l*10 + int(res[i]-'0')
		}
		re.res = res[i:]
		re.len = int64(l)
	case ':':
	case ' ':
	default:

	}
	return re
}
