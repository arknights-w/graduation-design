package engine

import (
	"Airfone/api/errorpb"
	"sync"
)

// 服务列表
type TopicMap struct {
	// 这把大锁是防止同名主题在
	// 同时创建，造成覆盖
	// 也就是说，只有在创建/删除主题时，会使用该锁
	// 注: 所有方法都没有加锁，应当在调用时依据业务来加锁
	// 		原因1: 每个方法加锁会导致反复的加解锁，产生大量开销
	// 		原因2: go原生不支持可重入锁，若无意间将锁重复lock，会导致严重bug
	sync.RWMutex
	topics map[string]*Topic
}

// 添加主题
//
//	加写锁
func (tm *TopicMap) AddTopic(name string) (*Topic, error) {
	tm.Lock()
	defer tm.Unlock()
	if _, ok := tm.topics[name]; ok {
		return nil, errorpb.ErrorInsertAlreadyExist("this topic is already existed")
	}
	topic := &Topic{
		running: make(map[int32]*Provider),
		pending: make(map[int32]*Provider),
	}
	tm.topics[name] = topic
	return topic, nil
}

// 移除主题
//
//	加写锁
func (tm *TopicMap) RemoveTopic(name string) {
	tm.Lock()
	defer tm.Unlock()
	delete(tm.topics, name)
}

// 获取主题
//
//	加读锁
func (tm *TopicMap) GetTopic(name string) (*Topic, error) {
	var (
		topic *Topic
		ok    bool
	)
	tm.RLock()
	defer tm.RUnlock()
	if topic, ok = tm.topics[name]; !ok {
		return nil, errorpb.ErrorSearchInvalid("no such a name for topic")
	}
	return topic, nil
}

// 获取主题，若主题不存在则创建主题
//
//	加写锁
func (tm *TopicMap) GetXTopic(name string) (*Topic, error) {
	var (
		topic *Topic
		ok    bool
	)
	tm.Lock()
	defer tm.Unlock()
	if topic, ok = tm.topics[name]; !ok {
		topic = &Topic{
			running: make(map[int32]*Provider),
			pending: make(map[int32]*Provider),
		}
		tm.topics[name] = topic
	}
	return topic, nil
}

type Topic struct {
	// 一般情况下，添加/修改/删除主题内单个provider时，
	// 又或者某个provider心跳超时未联系，会使用topic锁
	// 在读取时不需要上锁，更新单个provider的 keepalive 时也不需要上锁
	// 注: 所有方法都没有加锁，应当在调用时依据业务来加锁
	// 		原因1: 每个方法加锁会导致反复的加解锁，产生大量开销
	// 		原因2: go原生不支持可重入锁，若无意间将锁重复lock，会导致严重bug
	sync.RWMutex

	// 这个时间会决定消费者的依赖更新
	// 消费者列表中，每一个消费者都维护了一个更新时间
	// 心跳来了，程序会遍历消费者依赖的每一个topic
	// 当发现 topic 的 current 大于消费者的时间
	// 心跳就会向消费者反馈他需要更新的topic列表
	current int64               // 最新修改时间(纳秒，time.Now().UnixNano())
	running map[int32]*Provider // 正在运行的服务
	pending map[int32]*Provider // 暂时无法联系的服务
}

type Provider struct {
	// ip
	// port
	// schema
	// keepalive

	Schema    map[string]string // 元数据
	Keepalive int64             // 心跳时间(纳秒，time.Now().UnixNano())
	IP        int64             // ip
	Port      int16             // 端口
}

// 添加一个 provider 到 running 列表中
//
//	加写锁
func (t *Topic) AddProvider(now int64, id int32, provider *Provider) error {
	t.Lock()
	defer t.Unlock()
	if _, ok := t.running[id]; ok {
		return errorpb.ErrorInsertAlreadyExist("this provider is already existed")
	}
	t.running[id] = provider
	t.current = now
	return nil
}

// 获取正在运行的 provider
//
//	加读锁
func (t *Topic) GetRunningProvider(id int32) (*Provider, error) {
	t.RLock()
	defer t.RUnlock()
	if provider, ok := t.running[id]; !ok {
		return nil, errorpb.ErrorSearchInvalid("this provider is not exist in running list")
	} else {
		return provider, nil
	}
}

// 获取被 pending 的 provider
//
//	加读锁
func (t *Topic) GetPendingProvider(id int32) (*Provider, error) {
	t.RLock()
	defer t.RUnlock()
	if provider, ok := t.pending[id]; !ok {
		return nil, errorpb.ErrorSearchInvalid("this provider is not exist in pending list")
	} else {
		return provider, nil
	}
}

// 移除正在运行的 provider
//
//	加写锁
func (t *Topic) RemoveRunningProvider(now int64, id int32) {
	t.Lock()
	defer t.Unlock()
	delete(t.running, id)
	t.current = now
}

// 移除阻塞的 provider
//
//	加写锁
//	这个移除不会影响当前运行，因此不需要修改时间
func (t *Topic) RemovePendingProvider(id int32) {
	t.Lock()
	defer t.Unlock()
	delete(t.pending, id)
}

// 修改 provider 信息
//
//	加写锁
func (t *Topic) UpdateRunningProvider(now int64, id int32, provider *Provider) error {
	t.Lock()
	defer t.Unlock()
	if _, ok := t.running[id]; !ok {
		return errorpb.ErrorUpdateInvalid("this provider is not exist in running list")
	}
	t.running[id] = provider
	t.current = now
	return nil
}

// 复活
//
//	加写锁
//	让 pending 状态中的 provider 复活，回到 running 列表中
func (t *Topic) Resurrect(now int64, id int32) error {
	var (
		item *Provider
		ok   bool
	)
	t.Lock()
	defer t.Unlock()
	if item, ok = t.pending[id]; !ok {
		return errorpb.ErrorDeleteInvalid("this provider is not exist in pending list for resurrect")
	}
	t.running[id] = item
	delete(t.pending, id)
	t.current = now
	return nil
}

// 待裁决
//
//	加写锁
//	当长时间 provider 无响应时，将其从 running 转移到 pending
func (t *Topic) Pend(now int64, id int32) error {
	var (
		item *Provider
		ok   bool
	)
	t.Lock()
	defer t.Unlock()
	if item, ok = t.running[id]; !ok {
		return errorpb.ErrorDeleteInvalid("this provider is not exist in running list for pend")
	}
	t.pending[id] = item
	delete(t.running, id)
	t.current = now
	return nil
}
