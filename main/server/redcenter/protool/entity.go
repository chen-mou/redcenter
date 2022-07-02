package protool

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Header struct {
	Custom  map[string][]string
	Auth    string
	Length  int
	Path    string
	Query   map[string]string
	Method  string
	Alive   time.Duration
	locks   map[string]sync.Mutex `json:"-"`
	isClose bool                  `json:"-"`
}

type Body struct {
	isClose bool                  `json:"-"`
	locks   map[string]sync.Mutex `json:"-"`
	Data    map[string]interface{}
}

func read(data []byte) (string, error) {
	res := make([]byte, 1)
	start := false
	closed := false
	for i := 0; i < len(data) && !closed; i++ {
		if data[i] == '&' {
			start = true
			continue
		}
		if !start {
			continue
		}
		switch data[i] {
		case '-':
			closed = true
		case '\\':
			res = append(data, data[i])
			i++
		default:
			res = append(data, data[i])
		}
	}
	if !closed {
		return "", errors.New("错误数据结尾")
	}
	return string(res), nil
}

func (h *Header) Build(data []byte) error {
	res, err := read(data)
	if err != nil {
		return err
	}
	h.Custom = map[string][]string{}
	atrs := strings.Split(string(res), "\n")
	for _, value := range atrs {
		names := strings.Split(value, ":")
		switch names[1] {
		case "Auth", "auth", "AUTH":
			h.Auth = names[2]
		case "Content-Length", "content-length":
			var err error
			h.Length, err = strconv.Atoi(names[2])
			if err != nil {
				return errors.New("头属性Content-Length有误")
			}
		case "Path", "path":
			index := strings.Index(names[2], "?")
			if index == -1 {
				h.Path = names[2]
			} else {
				h.Path = names[2][:index]
				h.Query = map[string]string{}
				querys := strings.Split(names[2][index+1:], "&")
				for _, val := range querys {
					vals := strings.Split(val, "=")
					if len(vals) <= 2 {
						return errors.New("url有误")
					}
					h.Query[vals[0]] = vals[1]
				}
			}
		case "Method", "method":
			h.Method = names[2]
		case "AliveTime", "alivetime", "alive-time":
			{
				val, err := strconv.ParseInt(names[2], 10, 64)
				if err != nil {
					return errors.New("头属性Content-Length有误")
				}
				h.Alive = time.Duration(val)
			}
		default:
			h.Custom[names[0]] = strings.Split(names[1], ",")
		}
	}
	return nil
}

func (b *Body) Builder(data []byte) error {
	str, err := read(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(str), &b.Data)
	return err
}
