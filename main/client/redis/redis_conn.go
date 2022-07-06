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
	Conf    *Configure
	conn    net.Conn
	master  *Client
	lock    lockMap
	isClose bool
	index   int
}

type Configure struct {
	Host     string
	Port     string
	Password string
	Role     string
	Master   *Configure
}

type lockMap map[string]*sync.Mutex

func (client *Client) Open() error {
	client.lock = lockMap{
		"Close":     &sync.Mutex{},
		"PoolOrder": &sync.Mutex{},
	}
	client.lock["Close"].Lock()
	defer client.lock["Close"].Unlock()
	if client.isClose {
		client.isClose = false
	}
	conf := client.Conf
	con, err := net.Dial("tcp", conf.Host+":"+conf.Port)
	client.conn = con
	return err
}

func (client *Client) Close() error {
	if client.lock["Close"] == nil {
		return errors.New("连接未打开")
	}
	client.lock["Close"].Lock()
	defer client.lock["Close"].Unlock()
	if client.isClose {
		return errors.New("连接已关闭")
	}
	client.conn.Close()
	client.conn = nil
	client.isClose = true
	return nil
}

func (client *Client) Set(key string, value interface{}) (*ResCmd, error) {
	if client.Conf.Role == Follower {
		return nil, errors.New("从机无法写入")
	}
	var res *ResCmd
	switch value.(type) {
	case struct{}, map[string]string, map[string]interface{}:
		b, _ := json.Marshal(value)
		res = client.excute(fmt.Sprintf("set %s %s", key, "\""+strings.Replace(string(b), "\"", "\\\"", -1)+"\"")).(*ResCmd)
	case string:
		res = client.excute(fmt.Sprintf("set %s %s", key, value)).(*ResCmd)
	case bool:
		res = client.excute(fmt.Sprintf("set %s %t", key, value)).(*ResCmd)
	case int8, int, int64, int32, uint, uint8, uint32, uint64:
		res = client.excute(fmt.Sprintf("set %s %d", key, value)).(*ResCmd)
	case float32, float64:
		res = client.excute(fmt.Sprintf("set %s %f", key, value)).(*ResCmd)
	default:
		panic("类型:" + reflect.TypeOf(value).String() + "不支持")
	}
	return res, nil
}

func (client *Client) Get(key string) (*ResCmd, error) {
	return client.excute(fmt.Sprintf("get %s", key)).(*ResCmd), nil
}

func (client *Client) HSet(key string, value interface{}) error {
	if client.Conf.Role == Follower {
		return errors.New("从机无法写入")
	}
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
			tField := t.Field(i)
			name := tField.Name
			tag, ok := tField.Tag.Lookup("json")
			if ok {
				if tag == "-" {
					continue
				}
				name = tag
			}
			//fmt.Println(t.String())
		LOOP:
			switch field.Kind() {
			case reflect.Ptr:
				field = field.Elem()
				break LOOP
			case reflect.Struct:
				handlerStruct(prefix+"."+name, &field)
			case reflect.Map:
				ks := field.MapKeys()
				for _, k := range ks {
					keys = append(keys, prefix+"."+name, toString(k))
					argv = append(argv, "\""+toString(field.MapIndex(k))+"\"")
					l := len(keys)
					script += fmt.Sprintf(lua, l-1, l, len(argv))
				}
			case reflect.Array, reflect.Slice:
				keys = append(keys, prefix+"."+name)
				script += fmt.Sprintf("redis.call('LPUSH', KEYS[%d]", len(keys))
				for i := 0; i < field.Len(); i++ {
					argv = append(argv, toString(field.Index(i)))
					script += fmt.Sprintf(",ARGV[%d]", len(argv))
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
	client.excute(script)
	//client.excute("hget wabapp.Health Url")
	return nil
}

func (client *Client) HGetField(key, field string) (*ResCmd, error) {
	return client.excute(fmt.Sprintf("hget %s %s", key, field)).(*ResCmd), nil
}

func (client *Client) HSetField(key, field string, v interface{}) (*ResCmd, error) {
	if client.Conf.Role == Follower {
		return nil, errors.New("从机无法写入")
	}
	return client.excute(fmt.Sprintf("hset %s %s %s", key, field, interfaceTurnString(v))).(*ResCmd), nil
}

func (client *Client) HGet(key string, v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return errors.New("结构体不是指针没办法获取值")
	}
	val = val.Elem()
	var dfs func(prefix string, value *reflect.Value)
	var setValue func(field *reflect.Value, res Cmd)

	dfs = func(prefix string, value *reflect.Value) {
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			tField := value.Type().Field(i)
			name := tField.Name
			tag, ok := tField.Tag.Lookup("json")
			if ok {
				if tag == "-" {
					continue
				}
				name = tag
			}
			setValue = func(field *reflect.Value, res Cmd) {
				kind := field.Kind()
				switch {
				case kind >= 2 && kind <= 11:
					res := res.(*ResCmd)
					va, _ := strconv.ParseInt(res.res, 10, 64)
					field.SetInt(va)
				case kind == 1:
					res := res.(*ResCmd)
					va, _ := strconv.ParseBool(res.res)
					field.SetBool(va)
				case kind == 24:
					res := res.(*ResCmd)
					va := res.res
					field.SetString(va)
				case kind == 21:
					Map(field, res.(*ArrayCmd))
				case kind == 23 || kind == 17:
					Array(field, res)
				case kind == 25:
					dfs(prefix+"."+name, field)
				}
			}

			var res Cmd
			if field.Kind() == reflect.Map {
				res = client.excute(fmt.Sprintf("hgetall %s", prefix+"."+name))
			} else if field.Kind() == reflect.Struct {
				res = client.excute(fmt.Sprintf("hgetall %s", prefix+"."+name))
			} else {
				res = client.excute(fmt.Sprintf("hget %s %s", prefix, name))
			}
			setValue(&field, res)
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

func (client *Client) RoleInfo() (*ResCmd, error) {
	res := client.excute("info replication").(*ResCmd)
	if res.Error() != nil {
		return nil, res.Error()
	}
	//处理字符串结果
	infos := strings.Split(res.res, "\r\n")
	res.res = strings.Split(infos[1], ":")[1]
	res.len = int64(len(res.res))
	return res, nil
}

func (client *Client) LLen(key string) (int, error) {
	res := client.excute("LLen " + key)
	return res.Result().(int), res.Error()
}

func (client *Client) push(order, key string, vals ...interface{}) *NumberCmd {
	o := order + " " + key
	for _, v := range vals {
		o += " " + toString(reflect.ValueOf(v))
	}
	cmd := client.excute(o)
	return cmd.(*NumberCmd)
}

func (client *Client) LPush(key string, vals ...interface{}) *NumberCmd {
	return client.push("lpush", key, vals)
}

func (client *Client) RPush(key string, vals ...interface{}) *NumberCmd {
	return client.push("rpush", key, vals)
}

// BindLIndex 获取数组第index + 1 个位置的值v要是指针
func (client *Client) BindLIndex(key string, index int, v interface{}) error {
	res := client.excute(fmt.Sprintf("lindex %s %d", key, index))
	if res.Error() != nil {
		return res.Error()
	}
	json.Unmarshal([]byte(res.Result().(string)), v)
	return nil
}

func (client *Client) LIndex(key string, index int) *ResCmd {
	return client.excute(fmt.Sprintf("lindex %s %d", key, index)).(*ResCmd)
}

func (client *Client) excute(order string) Cmd {
	b := make([]byte, 1024)
	res := ""
	client.lock["PoolOrder"].Lock()
	n, err := client.conn.Write([]byte(order + "\r\n"))
	if err != nil {
		panic(err.Error())
	}
	n, err = client.conn.Read(b)
	if err != nil {
		panic(err.Error())
	}
	client.lock["PoolOrder"].Unlock()
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
	masterClient := &Client{
		Conf: master,
	}
	masterClient.Open()
	client.master = masterClient
	res := client.excute(fmt.Sprintf("slaveof %s %s", master.Host, master.Port))
	return res.Error()
}

func (client *Client) Ping() *ResCmd {
	return client.excute("ping").(*ResCmd)
}

func interfaceTurnString(val interface{}) string {
	v := reflect.ValueOf(val)
	return toString(v)
}

func toString(val reflect.Value) string {
	kind := val.Kind()
	return turnStringByKind(kind, val)
}

//将value类型转换为字符串
func turnStringByKind(kind reflect.Kind, val reflect.Value) string {
	//val 是整数
	if kind < 12 && kind > 1 {
		return strconv.FormatInt(val.Int(), 10)
	}
	//val 是浮点数
	if kind == 13 {
		return strconv.FormatFloat(val.Float(), 'f', 10, 32)
	} else if kind == 14 {
		return strconv.FormatFloat(val.Float(), 'f', 10, 64)
	}
	//val 是布尔类型
	if kind == 1 {
		return strconv.FormatBool(val.Bool())
	}
	//val 是字符串
	if kind == 24 {
		return val.String()
	}
	//val 是数组类型
	if kind == 23 || kind == 17 {
		first := val.Index(0)
		res := "[" + turnStringByKind(first.Kind(), first)
		for i := 1; i < val.Len(); i++ {
			v := val.Index(i)
			res += "," + turnStringByKind(first.Kind(), v)
		}
		res += "]"
		return res
	}
	if kind == reflect.Struct {
		b, _ := json.Marshal(val.Interface())
		return string(b)
	}
	if kind == reflect.Ptr {
		return toString(val.Elem())
	}
	return ""
}

func handlerResult(res string) Cmd {
	var re Cmd
	getNumber := func(s string) (string, int) {
		num := 0
		for s[0] != '\r' {
			num = num*10 + int(s[0]-'0')
			s = s[1:]
		}
		return s, num
	}
	//fmt.Println(res)
	switch res[0] {
	case '-':
		re = &ResCmd{
			err: res[1 : len(res)-2],
			len: 0,
		}
	case '+':
		re = &ResCmd{
			res: res[1 : len(res)-2],
		}
	case '$':
		isMinus := false
		if res[1] == '-' {
			isMinus = true
		}
		l := 0
		res, l = getNumber(res)
		if isMinus {
			l = -l
		}
		if l == -1 {
			re = &ResCmd{
				res: "",
				len: 0,
				err: "-1",
			}
			return re
		}
		re = &ResCmd{
			res: res[2 : len(res)-2],
			len: int64(l),
		}
	case ':':
	case ' ':
	case '*':
		res = res[1:]
		num := 0
		res, num = getNumber(res)
		res = res[2:]
		result := make([]string, num)
		for i := 0; i < num; i++ {
			res = res[1:]
			length := 0
			res, length = getNumber(res)
			result[i] = res[2 : length+2]
			res = res[length+4:]
		}
		re = &ArrayCmd{
			res: result,
			err: "",
			len: num,
		}
	default:

	}
	return re
}

func (client *Client) Execute(s string) Cmd {
	return client.excute(s)
}
