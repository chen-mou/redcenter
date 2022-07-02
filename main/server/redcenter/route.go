package redcenter

import (
	"RedisRegister/main/server/redcenter/protool"
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strings"
)

type Node struct {
	name       string
	typ        string
	middleWare []func(context2 *Context)
	handler    map[string]func(ctx *Context)
	children   map[string]*Node
}

func (n *Node) Handler(method, path string, handler func(ctx *Context)) *Node {
	index := strings.LastIndex(path, "/")

	if index == 0 {
		return n.Handler(method, path[1:], handler)
	}

	if index != -1 {
		node := n.Group(path[:index])
		node.Handler(method, path[index+1:], handler)
	} else {
		node, ok := n.children[path]
		if ok {
			node.handler[method] = handler
		} else {
			n.children[path] = &Node{
				name:       path,
				typ:        "handler",
				middleWare: make([]func(ctx *Context), 0),
				handler: map[string]func(ctx *Context){
					method: handler,
				},
			}
		}
	}
	return n
}

func (n *Node) Get(path string, handler func(ctx *Context)) *Node {
	n.Handler("get", path, handler)
	return n
}

func (n *Node) Post(path string, handler func(ctx *Context)) *Node {
	n.Handler("post", path, handler)
	return n
}

func (n *Node) MiddleWare(handler func(ctx *Context)) *Node {
	n.middleWare = append(n.middleWare, handler)
	return n
}

func (n *Node) Group(name string) *Node {
	index := strings.LastIndex(name, "/")
	if index == 0 {
		return n.Group(name[1:])
	}
	if index != -1 {
		n1, ok := n.children[name[:index]]
		if ok {
			return n.Group(name[index+1:])
		}
		n1 = n.Group(name[:index])
		n1.children[name[:index]] = n
		name = name[index+1:]
		return n.Group(name)
	} else {
		n1, ok := n.children[name]
		if ok {
			return n
		}
		n1 = &Node{
			name:       name,
			typ:        "group",
			middleWare: make([]func(ctx *Context), 0),
			handler:    map[string]func(ctx *Context){},
			children:   map[string]*Node{},
		}
		n.children[name] = n1
		return n
	}
}

type nodeList struct {
	index int
	now   *Node
	List  *nodeList
}

type Context struct {
	context.Context
	next   *nodeList
	now    *Node
	Header protool.Header
	Body   protool.Body
	conn   net.Conn
	atr    map[string]interface{}
}

type ResourceNotFindError struct {
	Path string
}

func (ctx *Context) make(body protool.Body, header protool.Header, conn net.Conn) {
	ctx.conn = conn
	ctx.Body = body
	ctx.Header = header

}

func (ctx *Context) BindJson(val interface{}) error {
	t := reflect.TypeOf(val)
	if t.Kind() != reflect.Ptr {
		return errors.New("非指针类型无法赋值")
	}
	b, _ := json.Marshal(ctx.Body.Data)
	err := json.Unmarshal(b, val)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *Context) QueryString(name string, def string) string {
	val, ok := ctx.Header.Query[name]
	if !ok {
		return def
	}
	return val
}

func (ctx *Context) Next() {
	nxt := ctx.next
	if nxt == nil {
		return
	}
	if nxt.index < len(nxt.now.middleWare) {
		nxt.index++
		nxt.now.middleWare[nxt.index-1](ctx)
	} else {
		if nxt.List == nil {
			nxt.now.handler[ctx.Header.Method](ctx)
		} else {
			ctx.next = nxt.List
			nxt = ctx.next
			nxt.index++
			nxt.now.middleWare[0](ctx)
		}
	}
}

func (ctx *Context) AbortWithJson(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	ctx.conn.Write(Successful("成功", b))
	return ctx.conn.Close()
}

func (ctx *Context) Abort() error {
	return ctx.conn.Close()
}

func (r ResourceNotFindError) Error() string {
	return r.Path + "不存在"
}

func (n Node) find(path string, method string) (*nodeList, error) {
	i := strings.Index(path, "/")
	var name string
	if i == 0 {
		return n.find(path[1:], method)
	}
	if i == -1 {
		name = path
	} else {
		name = path[:i]
	}
	ch, ok := n.children[name]
	if !ok || (i == -1 && (ch.typ == "group")) {
		return nil, ResourceNotFindError{
			Path: path,
		}
	}
	_, ok = n.handler[method]
	if !ok {
		return nil, ResourceNotFindError{
			Path: path,
		}
	}
	list := &nodeList{
		now:   ch,
		index: 0,
	}
	if i == -1 {
		list.List = nil
	} else {
		var err error
		list.List, err = ch.find(path[i+1:], method)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (n *nodeList) run(ctx *Context) {
	n.List.now.middleWare[0](ctx)
}
