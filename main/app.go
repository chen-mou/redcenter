package main

import (
	"RedisRegister/main/client/redis"
)

func main() {
	con := redis.Client{}
	con.Open(&redis.Configure{Host: "localhost", Port: "6379"})
	type Health struct {
		Status string
		Url    string
	}
	s := struct {
		Health Health
	}{
		Health: Health{
			Status: "Up",
			Url:    "localhost123456789",
		},
	}
	con.HSet("webapp", s)
}
