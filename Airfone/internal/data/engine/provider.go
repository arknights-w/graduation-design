package engine

import (
	"sync"
	"time"
)

// 服务列表
type TopicMap struct {
	// 这把大锁是防止同名主题在
	// 同时创建，造成覆盖
	// 也就是说，只有在创建/删除主题时，会使用该锁
	sync.RWMutex
	topics map[string]Topic
}

type Topic struct {
	// 一般情况下，添加/修改/删除主题内单个服务时，
	// 又或者某个服务心跳超时未联系，会使用topic锁
	// 在读取时不需要上锁，更新单个服务的 keepalive 时也不需要上锁
	sync.RWMutex
	current time.Time        // 最新修改时间
	running map[int]*Provider // 正在运行的服务
	pending map[int]*Provider // 暂时无法联系的服务
}

type Provider struct {
	// ip
	// port
	// schema
	// keepalive
}
