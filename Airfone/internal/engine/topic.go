package engine

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	DURATION_HEARTBEAT = 2 * time.Second // 心跳时间，每隔 DURATION_HEARTBEAT 轮询一次，客户端向服务器端发送一次心跳
	DURATION_VALID     = 3 * time.Second // 有效时间，当客户端超过 DURATION_VALID 未向服务器端发送心跳，即便在 running 队列中，也不纳入使用
	DURATION_PENDING   = 4 * time.Second // pending时间，若 service心跳 与当前时间相差 DURATION_PENDING 以上，则将其从running队列放到pending队列
	DURATION_DROPPED   = 6 * time.Second // dropping时间，若 pending队列中 service心跳 与当前时间相差 DURATION_DROPPED 则认为失联，将其状态置为 dropping 并删除
)

type Topic struct {
	// 一般情况下，添加/修改/删除主题内单个service时，
	// 又或者某个service心跳超时未联系，会使用 topic 写锁
	// 在读取 topic 时上读锁，更新单个service的 keepalive 时也不需要上锁
	sync.RWMutex

	running *ServiceMap // 正在运行的服务
	pending *ServiceMap // 暂时无法联系的服务
}

func NewTopic(log *log.Helper) *Topic {
	var (
		topic = &Topic{
			// current: now,
			running: NewServiceMap(),
			pending: NewServiceMap(),
		}
		ticker = time.NewTicker(DURATION_HEARTBEAT)
	)

	// todo: 每个 topic 加一个协程，用于异步处理失效数据
	// 如:
	// 1.定期将心跳失联的 service 放到 pending 队列中
	// 2.定期删除 pending 中过期的 service
	//
	// 每三秒执行一次，将 running 心跳间隔 DURATION_PENDING 以上的放入 pending
	// 将 pending 心跳间隔 DURATION_DROPPED 以上的删除
	go func() {
		log.Info("-----------------开启一个 topic 的异步协程-----------------")
		for {
			<-ticker.C
			log.Info("-----------------异步协程探测-----------------")
			now := time.Now()
			// 这里是处理 1
			topic.Lock()
			topic.running.Lock()
			topic.pending.Lock()
			if s, err := topic.running.BatchDeleteByCheckTime(now.UnixNano(), DURATION_PENDING); err != nil {
				log.Errorf("schedule Error: %s", err.Error())
			} else {
				for _, s2 := range s {
					s2.Status = HeartBeat_PENDING
				}
				if err = topic.pending.BatchAdd(now.UnixNano(), s); err != nil {
					log.Errorf("schedule Error: %s", err.Error())
				}
			}
			topic.Unlock()
			topic.running.Unlock()

			// 处理 2
			// todo
			_, err := topic.pending.BatchDeleteByCheckTime(now.UnixNano(), DURATION_DROPPED)
			if err != nil {
				log.Error("schedule Error: %s", err.Error())
			}
			topic.pending.Unlock()
		}
	}()

	return topic
}

// 添加一个 Service 到 running 列表中
func (t *Topic) AddRunningService(now int64, service *Service) (*Service, error) {
	if service.Status != HeartBeat_CHANGED {
		service.Status = HeartBeat_RUNNING
	}
	return t.running.Add(now, service)
}

// 添加一个 Service 到 pending 列表中
//
//	如果是将一个 running 队列中取出放入 pending 队列，应该使用 pend 方法，而不是使用该方法
func (t *Topic) AddPendingService(now int64, service *Service) (*Service, error) {
	service.Status = HeartBeat_PENDING
	return t.pending.Add(now, service)
}

// 获取正在运行的 Service
func (t *Topic) GetRunningService(id int32) (*Service, error) {
	return t.running.Get(id)
}

// 获取所有的 running 节点
//
//	running节点有俩要求:
//	1. 状态为HeartBeat_RUNNING
//	2. 心跳在有效期内(now - keepalive < DURATION_VALID)
//	注意 running 队列中有两种状态
//	1. changed，等待客户端回应成功，一旦回应成功则状态变更为 running
//	2. running，可用节点
func (t *Topic) GetAllRunningService(now int64) []*Service {
	t.running.RLock()
	defer t.running.RUnlock()
	var (
		list  = make([]*Service, 0, len(t.running.services))
		limit = now - int64(DURATION_VALID)
	)
	for _, s := range t.running.services {
		if s.keepalive > limit && s.Status == HeartBeat_RUNNING {
			list = append(list, s)
		}
	}
	return list
}

// 获取被 pending 的 Service
func (t *Topic) GetPendingService(id int32) (*Service, error) {
	return t.pending.Get(id)
}

// 获取该 id 的 service, 无论他在哪个列表中
func (t *Topic) GetService(id int32) (*Service, error) {
	var (
		serv *Service
		err  error
	)
	if serv, err = t.running.Get(id); err != nil {
		if serv, err = t.pending.Get(id); err != nil {
			return nil, err
		}
	}
	return serv, nil
}

// 移除正在运行的 Service
func (t *Topic) RemoveRunningService(now int64, id int32) (*Service, error) {
	s, err := t.running.Delete(now, id)
	if err != nil {
		return nil, err
	}
	s.Status = HeartBeat_DROPPED
	return s, nil
}

// 移除阻塞的 Service
func (t *Topic) RemovePendingService(now int64, id int32) (*Service, error) {
	s, err := t.pending.Delete(now, id)
	if err != nil {
		return nil, err
	}
	s.Status = HeartBeat_DROPPED
	return s, nil
}

// 移除该 id 的 service, 无论他在哪个列表中
func (t *Topic) RemoveService(now int64, id int32) (*Service, error) {
	var (
		serv *Service
		err  error
	)
	if serv, err = t.RemoveRunningService(now, id); err != nil {
		if serv, err = t.RemovePendingService(now, id); err != nil {
			return nil, err
		}
	}
	return serv, nil
}

// 复活
//
//	思路: 将 service 从 pending 队列中删除，
//	再将其添加到 running 队列，若失败则重新将 service 加回到 pending 队列
//	最终修改时间，返回
func (t *Topic) Resurrect(now int64, id int32) (*Service, error) {
	var (
		service *Service
		err     error
	)
	t.Lock()
	defer t.Unlock()
	if service, err = t.pending.Delete(now, id); err != nil {
		return nil, err
	}
	// 这里修改 service 的状态，从 pending 到 running
	if service.Status != HeartBeat_CHANGED {
		service.Status = HeartBeat_RUNNING
	}
	if service, err = t.running.Add(now, service); err != nil {
		return nil, err
	}
	return service, nil
}

// 复活2
//
//	若原本 id 就在 running 队列，则不做任何操作
//	若在 pending 队列则与 Resurrect 逻辑一致
func (t *Topic) ResurrectX(now int64, id int32) (*Service, error) {
	var (
		service *Service
		err     error
	)
	if service, err = t.running.Get(id); err != nil {
		if service, err = t.Resurrect(now, id); err != nil {
			return nil, err
		}
	} else {
		if service.Status != HeartBeat_CHANGED {
			service.Status = HeartBeat_RUNNING
		}
		service.keepalive = now
	}
	return service, nil
}

// 待裁决
//
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
	service.Status = HeartBeat_PENDING
	if _, err = t.pending.Add(now, service); err != nil {
		return err
	}
	return nil
}

// 待裁决2
//
//	若原本 id 就在 pending 队列，则不做任何操作
//	当长时间 Service 无响应时，将其从 running 转移到 pending
func (t *Topic) PendX(now int64, id int32) error {
	if _, err := t.pending.Get(id); err != nil {
		return t.Pend(now, id)
	}
	return nil
}

func (t *Topic) Conform(now int64, id int32) error {
	var (
		serv *Service
		err  error
	)
	if serv, err = t.running.Get(id); err != nil {
		return err
	}
	serv.keepalive = now
	serv.Status = HeartBeat_RUNNING
	return nil
}
