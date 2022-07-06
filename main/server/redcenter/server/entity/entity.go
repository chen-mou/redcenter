package entity

import "time"

type Info struct {
	Health string

	Refresh string

	Page string

	//管理页面
	Admin string
}

// Balance 负载均衡有三种方式 1.轮询 2.权重随机 3.IP哈希
// Hash 用于存Hash值由服务器算出
// Index 当前轮询到的下标
// Weight 权重
type Balance struct {
	Name string

	Index int

	Weight int

	Hash int
}

type ServerInfo struct {
	Host string

	Port string

	ServiceId int

	Status string

	lastBeatTime *time.Time

	WebInfo Info
}

// RegisterInfo 服务可以同名但同名的服务ServiceId不能相同相同的服务名在获取时会返回所有名字在调用时会使用特定的负载均衡办法调用
type RegisterInfo struct {
	Name string

	Services []ServerInfo

	Balance Balance
}
