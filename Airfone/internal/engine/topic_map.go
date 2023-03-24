package engine

import (
	"Airfone/api/errorpb"
)

// TopicMap about
//	这里面其实也全是 data 的方法，
//	不过是私有方法，同时需要加锁，
//	而 data.go 中则是调用这些私有方法，同时也不关心锁

// 添加主题
//
//	加写锁
func (tm *Data) addTopic(name string) (*Topic, error) {
	tm.Lock()
	defer tm.Unlock()
	if _, ok := tm.topics[name]; ok {
		return nil, errorpb.ErrorInsertAlreadyExist("this topic is already existed")
	}
	topic := NewTopic(tm.log)
	tm.topics[name] = topic
	return topic, nil
}

// 移除主题
//
//	加写锁
func (tm *Data) removeTopic(name string) {
	tm.Lock()
	defer tm.Unlock()
	delete(tm.topics, name)
}

// 获取主题
//
//	加读锁
func (tm *Data) getTopic(name string) (*Topic, error) {
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
func (tm *Data) getXTopic(name string) (*Topic, error) {
	var (
		topic *Topic
		ok    bool
	)
	tm.Lock()
	defer tm.Unlock()
	if topic, ok = tm.topics[name]; !ok {
		topic = NewTopic(tm.log)
		tm.topics[name] = topic
	}
	return topic, nil
}
