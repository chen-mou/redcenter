package protool

import (
	"errors"
	"net"
	"sync"
)

type Head struct {
	Custom   map[string][]string
	Auth     string
	locks    map[string]sync.Mutex `json:"-"`
	isClose  bool                  `json:"-"`
	net.Conn `json:"-"`
}

type Body struct {
	isClose  bool                  `json:"-"`
	locks    map[string]sync.Mutex `json:"-"`
	net.Conn `json:"-"`
}

func (b *Body) Open() error {
	if b.isClose == true {
		return errors.New("连接关闭后无法打开")
	}
	return nil
}

func (b *Body) Close() error {
	return nil
}

func (h *Head) Open() error {
	if h.isClose == true {
		return errors.New("连接关闭后无法打开")
	}
	return nil
}

func (h *Head) Close() error {
	return nil
}
