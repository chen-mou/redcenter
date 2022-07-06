package service

import (
	"RedisRegister/main/client/redis"
	"RedisRegister/main/server/redcenter/server/entity"
)

var Default = struct {
	GetByName func(string) []entity.ServerInfo
	Register  func(info entity.RegisterInfo) error
	HeartBeat func(string, int) error
}{
	GetByName: getByName,
}

func getByName(name string) []entity.ServerInfo {
	redis.Pool.Submit(3000, func(conn *redis.Client) interface{} {
		l, err := conn.LLen(name)
	}, func() {

	})
}

func register(info entity.RegisterInfo) error {

}

func HeartBeat(name string, id int) error {

}
