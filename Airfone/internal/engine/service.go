package engine

type HeartBeatType uint8

const (
	HeartBeat_RUNNING HeartBeatType = iota // 一切正常
	HeartBeat_CHANGED                      // 部分依赖内容已改变，需要重新拉取
	HeartBeat_PENDING                      // 上游服务已崩坏，依赖服务不可用，当前服务需要阻塞
	HeartBeat_DROPPED                      // 自身因延迟已被服务端删除，需要重新注册
)

type Service struct {
	Rely      []*Rely       // 依赖项 map[topic_name]service
	Schema    []*Schema     // 元数据
	keepalive int64         // [内部属性]心跳时间(纳秒，time.Now().UnixNano())
	IP        string        // ip
	ID        int32         // 唯一标识符
	Port      uint16        // 端口
	Status    HeartBeatType // 服务状态, 这个字段通常在 topic 层被操纵，dropping 能够在 service_map 层赋值
}

type HeartBeat struct {
	Rely   []*Rely       // 依赖
	Topic  string        // 主题
	ID     int32         // ID
	Status HeartBeatType // 状态
}

type Rely struct {
	Keepalive *int64         // 心跳时间，取的是所依赖的服务的心跳地址，可以更简单的判断
	Status    *HeartBeatType // 服务状态，也是取所依赖服务的状态的地址
	Topic     string         // 主题
	IP        string         // ip
	ID        int32          // 唯一标识符
	Port      uint16         // 端口
}

type Schema struct {
	Title   string // 元数据标题
	Content string // 内容
}
