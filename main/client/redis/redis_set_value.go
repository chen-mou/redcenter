package redis

import (
	"reflect"
	"strconv"
	"strings"
)

func Atov(s string, t reflect.Type) reflect.Value {
	kind := t.Kind()
	switch {
	case kind == 24 ||
		(kind >= 1 && kind <= 11) ||
		(kind >= 13 && kind <= 16):
		return reflect.ValueOf(AtoInter(s, t))
	case kind == 23 || kind == 17:
		return array(s, t)
	}
	panic("未编写的类型" + kind.String())
}

func AtoInter(s string, t reflect.Type) interface{} {
	kind := t.Kind()
	switch {
	case kind == 1:
		b, _ := strconv.ParseBool(s)
		return b
	case kind >= 2 && kind <= 6:
		n, _ := strconv.ParseInt(s, 10, (int(kind)-2)*8)
		return n
	case kind >= 7 && kind <= 11:
		n, _ := strconv.ParseUint(s, 10, (int(kind)-7)*8)
		return n
	case kind == 13 || kind == 14:
		n, _ := strconv.ParseFloat(s, (int(kind)-12)*32)
		return n
	case kind == 15 || kind == 16:
		n, _ := strconv.ParseComplex(s, (int(kind)-14)*64)
		return n
	case kind == 12:
		return AtoInter(s, t.Elem())
	case kind == 23 || kind == 17:
		return array(s, t).Interface()
	case kind == 24:
		return s
	}
	panic("未编写的类型" + kind.String())
}

//将string 转换为array类型
func array(s string, t reflect.Type) reflect.Value {
	s = s[1 : len(s)-1]
	f := func(left, right byte) []string {
		num := 0
		start := 0
		strs := make([]string, 0)
		for ; start < len(s); start++ {
			if s[start] == left {
				num++
			}
			if s[start] == right {
				num--
			}
			if num == 0 {
				strs = append(strs, s[:start])
				start++
			}
		}
		return strs
	}
	set := func(strs []string, t reflect.Type) reflect.Value {
		res := reflect.MakeSlice(t, len(strs), len(strs)+3)
		for i, v := range strs {
			res.Index(i).Set(Atov(v, t.Elem()))
		}
		return res
	}
	tElem := t.Elem()
	var r reflect.Value
	switch tElem.Kind() {
	case 21, 25:
		strs := f('{', '}')
		r = set(strs, tElem)
	case 23:
		strs := f('[', ']')
		r = set(strs, tElem)
	default:
		strs := strings.Split(s, ",")
		r = set(strs, t)
	}
	return r
}

func Array(field *reflect.Value, res Cmd) {
	switch res.(type) {
	case *ResCmd:
		field.Set(array(res.Result().(string), field.Type()))
	case *ArrayCmd:
		t := field.Type().Elem()
		var result reflect.Value
		if t.Kind() != reflect.String {
			l := len(res.Result().([]string))
			result = reflect.MakeSlice(t, l, l+3)
			for i := 0; i < l; i++ {
				result.Index(i).Set(Atov(res.Result().([]string)[i], t))
			}
		} else {
			result = reflect.ValueOf(res.Result())
		}
		field.Set(result)
	}
}

func Map(field *reflect.Value, res *ArrayCmd) {
	fType := field.Type()
	kType := fType.Key()
	eType := fType.Elem()
	m := reflect.MakeMap(fType)
	for in := 1; in < len(res.res); in += 2 {
		m.SetMapIndex(Atov(res.res[in-1], kType),
			Atov(res.res[in], eType))
	}
	field.Set(m)
}
