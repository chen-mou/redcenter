package protool

import (
	"sync"
)

type Head struct {
	Custom  map[string][]string
	Auth    string
	Length  int
	locks   map[string]sync.Mutex `json:"-"`
	isClose bool                  `json:"-"`
}

type Body struct {
	isClose bool                  `json:"-"`
	locks   map[string]sync.Mutex `json:"-"`
	Data    interface{}
}
