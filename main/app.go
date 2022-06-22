package main

import (
	"fmt"
	"reflect"
)

func main() {
	//con := redis.RedisClient{}
	//con.Open(&redis.Configure{Host: "localhost", Port: "6379"})
	//fmt.Println(con.Set("1", map[string]interface{}{
	//	"a": "b",
	//	"c": "d",
	//}).Result())
	//fmt.Println(con.Get("1").Result())
	m := map[string]string{
		"a": "b",
		"c": "d",
		"e": "f",
	}
	v := reflect.ValueOf(m)
	keys := v.MapKeys()
	for _, val := range keys {
		fmt.Println(val.String() + ":" + v.MapIndex(val).String())
	}
}
