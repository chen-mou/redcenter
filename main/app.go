package main

import (
	"RedisRegister/main/connect"
	"fmt"
)

func main() {
	con := connect.RedisClient{}
	con.Open(&connect.Configure{Host: "localhost", Port: "6379"})
	fmt.Println(con.Send("get key"))
}
