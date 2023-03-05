package engine

type Action int8

const (
	TURN_TO_PENDING     Action = iota // 将 provider 移动到 pending 队列
	DELETE_FROM_PENDING               // 将 pending 中过期的 provider 删除
)

type Schedule struct {
	Content map[string][]int32 // string -> topic_name, i32 -> provider_id
	Action  Action             // 行为
}
