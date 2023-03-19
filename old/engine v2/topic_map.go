package engine

import (
	"Airfone/api/errorpb"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// 服务列表
type TopicMap struct {
	// 这把大锁是防止同名主题在
	// 同时创建，造成覆盖
	// 也就是说，只有在创建/删除主题时，会使用该锁
	sync.RWMutex
	log    *log.Helper
	topics map[string]*Topic
}

func NewTopicMap(logger *log.Helper) *TopicMap {
	return &TopicMap{
		topics: make(map[string]*Topic),
		log:    logger,
	}
}

// 添加主题
//
//	加写锁
func (tm *TopicMap) AddTopic(name string) (*Topic, error) {
	tm.Lock()
	defer tm.Unlock()
	now := time.Now()
	if _, ok := tm.topics[name]; ok {
		return nil, errorpb.ErrorInsertAlreadyExist("this topic is already existed")
	}
	topic := NewTopic(now.UnixNano(), tm.log)
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
		now := time.Now()
		topic = NewTopic(now.UnixNano(), tm.log)
		tm.topics[name] = topic
	}
	return topic, nil
}
