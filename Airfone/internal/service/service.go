package service

import "github.com/google/wire"

// 服务注册: 
// 	将自身服务的对应信息发送至该服务中心注册，进行存储
// 	注册信息:
// 		1. ip + port 必须
// 		2. 映射 服务名
// 		3. 服务元数据: 服务版本，服务唯一标识
// 		4. 心跳数据: 包含服务信息，如服务当前所在机房，机器信息等
// 
// 	服务元数据: 
//		1. 服务 id: 如机房+机器+时间戳+序列号
//		2. 版本: 灰度发布功能，
// 		3. 注册时间: 时间戳
//
//	高级数据:
//		1. 健康检查: 检查次数，检查健康时间
//		2. 区域信息: zone，region
//		3. 集群同步信息:  

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewAirfoneService,
)
