package controller

import (
	"RedisRegister/main/server/redcenter"
	"RedisRegister/main/server/redcenter/server/entity"
)

func Register(s *redcenter.Server) {
	s.Group("/api").
		Post("/register", register).
		Get("/getList", getList).
		Get("/getServer", getByName)
}

func register(ctx *redcenter.Context) {
	data := &entity.RegisterInfo{}
	err := ctx.BindJson(data)
	if err != nil {
		ctx.AbortWithError(err)
	}
}

func getList(ctx *redcenter.Context) {

}

func getByName(ctx *redcenter.Context) {}
