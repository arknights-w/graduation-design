package engine

import (
	"Airfone/api/errorpb"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
)

// 依赖列表
// int 是消费者的id
// rely中是被消费者的信息，以及更新时间
// 心跳来时，就从这里查询依赖信息
// 如果信息不一致，则让客户端再拉取一次
type ConsumerMap struct {
	consumers map[int]*Rely
}

type Rely struct {
	topics  []string
	current time.Time // 最新修改时间
}

// 增加一个消费者
func (rm *ConsumerMap) Add(id int, topics []string) {
	rm.consumers[id] = &Rely{
		topics:  topics,
		current: time.Now(),
	}
}

// 移除一个消费者
func (rm *ConsumerMap) Remove(id int) {
	delete(rm.consumers, id)
}

// 修改该消费者的依赖项
func (rm *ConsumerMap) Update(id int, topics []string) error {
	var (
		r  *Rely
		ok bool
	)
	if r, ok = rm.consumers[id]; !ok {
		return errors.NotFound(errorpb.ErrorReason_INVALID_ID.String(), "no such a id for consumers")
	}

	r.topics = topics
	r.current = time.Now()
	return nil
}

// 消费者的依赖变更时间
func (rm *ConsumerMap) UpdateTime(id int, now time.Time) error {
	var (
		r  *Rely
		ok bool
	)
	if r, ok = rm.consumers[id]; !ok {
		return errors.NotFound(errorpb.ErrorReason_INVALID_ID.String(), "no such a id for consumers")
	}

	r.current = now
	return nil
}

// 获取消费者当前信息
func (rm *ConsumerMap) GetRelyTopics(id int) ([]string, time.Time, error) {
	var (
		r  *Rely
		ok bool
	)
	if r, ok = rm.consumers[id]; !ok {
		return nil, time.Time{},
			errors.NotFound(errorpb.ErrorReason_INVALID_ID.String(), "no such a id for consumers")
	}

	return r.topics, r.current, nil
}
