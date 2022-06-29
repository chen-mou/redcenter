package protool

import (
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
