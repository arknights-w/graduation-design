package engine

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	THREE_SECOND = 3 * time.Second
	SIX_SECOND   = 6 * time.Second
)

type Topic struct {
	// 一般情况下，添加/修改/删除主题内单个service时，
	// 又或者某个service心跳超时未联系，会使用 topic 写锁
	// 在读取 topic 时上读锁，更新单个service的 keepalive 时也不需要上锁
	sync.RWMutex

	// 这个时间会决定消费者的依赖更新
	// 消费者列表中，每一个消费者都维护了一个更新时间
	// 心跳来了，程序会遍历消费者依赖的每一个topic
	// 当发现 topic 的 current 大于消费者的时间
	// 心跳就会向消费者反馈他需要更新的topic列表
	current int64       // 最新修改时间(纳秒，time.Now().UnixNano())
	running *ServiceMap // 正在运行的服务
	pending *ServiceMap // 暂时无法联系的服务
}

func NewTopic(now int64, log *log.Helper) *Topic {
	var (
		topic = &Topic{
			current: now,
			running: NewServiceMap(),
			pending: NewServiceMap(),
		}
		ticker = time.NewTicker(THREE_SECOND)
	)

	// todo: 每个 topic 加一个协程，用于异步处理失效数据
	// 如:
	// 1.定期将心跳失联的 service 放到 pending 队列中
	// 2.定期删除 pending 中过期的 service
	go func() {
		for {
			<-ticker.C
			now := time.Now()
			// 这里是处理 1
			topic.Lock()
			if s, err := topic.running.BatchDeleteByCheckTime(now.UnixNano(), THREE_SECOND); err != nil {
				log.Errorf("schedule Error: %s", err.Error())
			} else {
				if err = topic.pending.BatchAdd(now.UnixNano(), s); err != nil {
					log.Errorf("schedule Error: %s", err.Error())
				}
			}
			topic.Unlock()

			// 处理 2
			// todo
		}
	}()

	return topic
}

// 添加一个 Service 到 running 列表中
func (t *Topic) AddService(now int64, id int32, service *Service) error {
	service.Status = RUNNING
	t.running.Add(now, id, service)
	t.current = now
	return nil
}

// 获取正在运行的 Service
func (t *Topic) GetRunningService(id int32) (*Service, error) {
	return t.running.Get(id)
}

// 获取被 pending 的 Service
func (t *Topic) GetPendingService(id int32) (*Service, error) {
	return t.pending.Get(id)
}

// 移除正在运行的 Service
func (t *Topic) RemoveRunningService(now int64, id int32) (*Service, error) {
	s, err := t.running.Delete(now, id)
	if err != nil {
		return nil, err
	}
	t.current = now
	return s, nil
}

// 移除阻塞的 Service
//
//	这个移除不会影响当前运行，因此不需要修改 topic 时间
func (t *Topic) RemovePendingService(now int64, id int32) (*Service, error) {
	s, err := t.running.Delete(now, id)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// 修改 Service 信息
func (t *Topic) UpdateRunningService(now int64, id int32, service *Service) (*Service, error) {
	s, err := t.running.Update(now, id, service)
	if err != nil {
		return nil, err
	}
	t.current = now
	return s, nil
}

// 复活
//
//	加写锁
//	思路: 将 service 从 pending 队列中删除，
//	再将其添加到 running 队列，若失败则重新将 service 加回到 pending 队列
//	最终修改时间，返回
func (t *Topic) Resurrect(now int64, id int32) error {
	var (
		service *Service
		err     error
	)
	t.Lock()
	defer t.Unlock()
	if service, err = t.pending.Delete(now, id); err != nil {
		return err
	}
	// 这里修改 service 的状态，从 pending 到 running
	service.Status = RUNNING
	if service, err = t.running.Add(now, id, service); err != nil {
		service.Status = PENDING
		if _, err := t.pending.Add(now, id, service); err != nil {
			return err
		}
		return err
	}
	t.current = now
	return nil
}

// 待裁决
//
//	加写锁
//	当长时间 Service 无响应时，将其从 running 转移到 pending
func (t *Topic) Pend(now int64, id int32) error {
	var (
		service *Service
		err     error
	)
	t.Lock()
	defer t.Unlock()

	if service, err = t.running.Delete(now, id); err != nil {
		return err
	}
	// 这里修改 service 的状态，从 running 到 pending
	service.Status = PENDING
	if service, err = t.pending.Add(now, id, service); err != nil {
		service.Status = RUNNING
		if _, err := t.running.Add(now, id, service); err != nil {
			return err
		}
		return err
	}
	t.current = now
	return nil
}
