package engine

import (
	"Airfone/api/errorpb"
	"sync"
)

// 依赖列表
//
//	int32 是消费者的id
//	rely中是被消费者的信息，以及更新时间
//	心跳来时，就从这里查询依赖信息
//	如果信息不一致，则让客户端再拉取一次
type ConsumerMap struct {
	sync.RWMutex
	consumers map[int32]*Rely
}

type Rely struct {
	current int64 // 最新修改时间(纳秒，time.Now().UnixNano())
	Topics  []string
}

// 增加一个消费者
//
//	加写锁
func (rm *ConsumerMap) Add(now int64, id int32, rely *Rely) error {
	rm.Lock()
	defer rm.Unlock()
	if _, ok := rm.consumers[id]; ok {
		return errorpb.ErrorInsertAlreadyExist("this id is already existed")
	}
	rely.current = now
	rm.consumers[id] = rely
	return nil
}

// 移除一个消费者
//
//	加写锁
func (rm *ConsumerMap) Remove(id int32) {
	rm.Lock()
	defer rm.Unlock()
	delete(rm.consumers, id)
}

// 修改该消费者的依赖项
//
//	加写锁
func (rm *ConsumerMap) Update(now int64, id int32, topics []string) error {
	var (
		r  *Rely
		ok bool
	)
	rm.Lock()
	defer rm.Unlock()

	if r, ok = rm.consumers[id]; !ok {
		return errorpb.ErrorUpdateInvalid("no such a id for consumers")
	}
	r.Topics = topics
	r.current = now
	return nil
}

// 消费者的依赖变更时间
//
//	加写锁
//	这个是消费者依赖的topic没有改变的情况下，
//	消费者所依赖的 provider 更新了，对应更新
//	依赖他的 cumsumer 的更新时间
func (rm *ConsumerMap) UpdateTime(now int64, id int32) error {
	var (
		r  *Rely
		ok bool
	)
	rm.Lock()
	defer rm.Unlock()

	if r, ok = rm.consumers[id]; !ok {
		return errorpb.ErrorUpdateInvalid("no such a id for consumers")
	}
	r.current = now
	return nil
}

// 获取消费者当前信息
//
//	加读锁
//	这里不返回指针，是为了防止被外部修改，导致逻辑混乱
func (rm *ConsumerMap) GetRelyTopics(id int32) (Rely, error) {
	var (
		r  *Rely
		ok bool
	)
	rm.Lock()
	defer rm.Unlock()

	if r, ok = rm.consumers[id]; !ok {
		return Rely{},
			errorpb.ErrorSearchInvalid("no such a id for consumers")
	}
	return *r, nil
}
