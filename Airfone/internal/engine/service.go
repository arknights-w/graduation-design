package engine

import "sync"

type SeriveStatus uint8

const (
	RUNNING  SeriveStatus = iota // 正常运行
	PENDING                      // 被阻塞，短暂未响应，保留的
	DROPPING                     // 被确认删除的，遗弃的
)

type Service struct {
	// ip
	// port
	// schema
	// keepalive
	// rely
	sync.RWMutex                     // 这个锁主要是为了 rely 和 schema 准备的，map 不允许并发写
	Rely         map[string]*Service // 依赖项 map[topic_name]service
	Schema       map[string]string   // 元数据
	keepalive    int64               // [内部属性]心跳时间(纳秒，time.Now().UnixNano())
	IP           string              // ip
	ID           int32               // 唯一标识符
	Port         uint16              // 端口
	Status       SeriveStatus        // 服务状态, 这个字段通常在 topic 层被操纵
}
