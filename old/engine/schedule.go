package engine

import (
	"encoding/json"
	"time"
)

// 这个文件中存放的是延时任务和定时任务的相关函数和类型

// 延时任务类型，目前只有一个 TURN_TO_PENDING，不排除有新增的可能，先暂时这样
type TaskAction int8

const (
	TURN_TO_PENDING TaskAction = iota // 将 provider 移动到 pending 队列
	// DELETE_FROM_PENDING                   // 将 pending 中过期的 provider 删除
)

const (
	TEN_SECOND   = 10000000000 // 10秒
	SIX_SECOND   = 6000000000  // 6秒
	FIVE_SECOND  = 5000000000  // 5秒
	THREE_SECOND = 3000000000  // 3秒
	ONE_SECOND   = 1000000000  // 1秒
)

type Task struct {
	Content map[string][]int32 // string -> topic_name, i32 -> provider_id
	Action  TaskAction         // 行为
}

// 将失效 provider 放到 pending 队列中
//
//	在方法内部处理错误，因为无法再继续外抛了，同时日志记录也会比较复杂
func turn_to_pending(data *Data, task *Task) {
	var (
		now = time.Now().UnixNano()
	)
	for k, v := range task.Content {
		data.providers.RLock()
		topic, err := data.providers.GetTopic(k)
		data.providers.RUnlock()
		if err != nil {
			data.log.Errorf("%s: %s", "TURN_TO_PENDING", err.Error())
			return
		}
		topic.Lock()
		for _, id := range v {
			err = topic.Pend(now, id)
			if err != nil {
				data.log.Errorf("%s: %s", "TURN_TO_PENDING", err.Error())
				topic.Unlock()
				return
			}
		}
		topic.Unlock()
	}
	// 日志: 操作类型: 内容
	if b, err := json.Marshal(task.Content); err != nil {
		data.log.Errorf("%s: %s", "TURN_TO_PENDING", err.Error())
	} else {
		data.log.Infof("%s: %s", "TURN_TO_PENDING", b)
	}
}

// 将超时 pending provider 删除
//
//	todo: 这个逻辑应该是更加复杂的，在删除 provider 的同时，
//	这个 id 可能同时也存在于 consumers 中，需要删除该 consumer，
//	若有 session 之类的状态存储，也应该清理
//
//	todo: for 循环可能会存在并发错误问题
// 
//	在方法内部处理错误，因为无法再继续外抛了，同时日志记录也会比较复杂
func drop_after_pending(data *Data) {
	var (
		now = time.Now().UnixNano()
	)
	for _, topic := range data.providers.topics {
		topic.Lock()
		for k, v := range topic.pending {
			if v.Keepalive < now-SIX_SECOND {
				topic.RemovePendingProvider(k)
			}
		}
		topic.Unlock()
	}
}
