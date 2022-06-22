package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const (
	Master   = "master"
	Follower = "follower"
)

type Client struct {
	Conf   *Configure
	conn   net.Conn
	master *Client
	lock   lockMap
}

type Configure struct {
	Host     string
	Port     string
	Password string
	Role     string
	Master   *Configure
}

type lockMap map[string]*sync.Mutex

func (client *Client) Open(conf *Configure) {
	con, err := net.Dial("tcp", conf.Host+":"+conf.Port)
	if err != nil {
		panic(err)
	}
	client.conn = con
	client.lock = lockMap{}
}

func (client *Client) Set(key string, value interface{}) *ResCmd {
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

func (client *Client) Get(key string) *ResCmd {
	return client.send(fmt.Sprintf("get %s", key))
}

func (client *Client) HSet(key string, value interface{}) error {
	var handlerStruct func(prefix string, value *reflect.Value)
	lua := "redis.call('hset', KEYS[%d], KEYS[%d], ARGV[%d]);"
	keys := make([]string, 0)
	argv := make([]string, 0)
	script := "eval \""
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	handlerStruct = func(prefix string, value *reflect.Value) {
		t := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			name := t.Field(i).Name
			//fmt.Println(t.String())
			switch field.Kind().String() {
			case "struct":
				handlerStruct(prefix+"."+name, &field)
			case "map":
				ks := field.MapKeys()
				for _, k := range ks {
					keys = append(keys, prefix, toString(k))
					argv = append(argv, toString(field.MapIndex(k)))
					l := len(keys)
					script += fmt.Sprintf(lua, l-1, l, len(argv))
				}
			default:
				keys = append(keys, prefix, name)
				argv = append(argv, toString(field))
				l := len(keys)
				script += fmt.Sprintf(lua, l-1, l, len(argv))
			}
		}
	}
	_, ok := client.lock[key]
	if !ok {
		client.lock[key] = &sync.Mutex{}
	}
	client.lock[key].Lock()
	defer client.lock[key].Unlock()
	handlerStruct(key, &v)
	script = "" + script + "return 'ok';\" " + strconv.Itoa(len(keys))
	for _, i := range keys {
		script += " " + i
	}
	for _, i := range argv {
		script += " " + i
	}
	client.send(script)
	//client.send("hget wabapp.Health Url")
	return nil
}

func (client *Client) HGetField(key, field string) *ResCmd {
	return nil
}

func (client *Client) HSetField(key, field string, v interface{}) *ResCmd {
	return client.send(fmt.Sprintf("hset %s %s %s", key, field, interfaceTurnString(v)))
}

func (client *Client) HGet(key string, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return errors.New("结构体不是指针没办法获取值")
	}
	val = val.Elem()
	var dfs func(prefix string, value *reflect.Value)
	dfs = func(prefix string, value *reflect.Value) {
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			kind := field.Kind()
			name := field.Type().Field(i).Name
			res := client.send(fmt.Sprintf("hget %s %s", prefix, name))
			if kind >= 2 && kind <= 11 {
				va, _ := strconv.ParseInt(res.res, 10, 64)
				field.SetInt(va)
			}
			if kind == 1 {
				va, _ := strconv.ParseBool(res.res)
				field.SetBool(va)
			}
			if kind == 24 {
				va := res.res
				field.SetString(va)
			}
			if kind == 25 {
				dfs(prefix+"."+name, &field)
			}
		}
	}
	_, ok := client.lock[key]
	if !ok {
		client.lock[key] = &sync.Mutex{}
	}
	client.lock[key].Lock()
	defer client.lock[key].Unlock()
	dfs(key, &val)
	return nil
}

func (client *Client) send(order string) *ResCmd {
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

func (client *Client) ConnectMaster(master *Configure) error {
	masterClient := &Client{}
	masterClient.Open(master)
	client.master = masterClient
	res := client.send(fmt.Sprintf("slaveof %s %s", master.Host, master.Port))
	return res.Error()
}

func interfaceTurnString(val interface{}) string {
	v := reflect.ValueOf(val)
	return toString(v)
}

func toString(val reflect.Value) string {
	kind := val.Kind()
	return turnStringByKind(kind, val)
}

func turnStringByKind(kind reflect.Kind, val reflect.Value) string {
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
		re.res = res[i+2 : len(res)-2]
		re.len = int64(l)
	case ':':
	case ' ':
	default:

	}
	return re
}
