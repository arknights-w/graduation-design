package engine

import (
	"Airfone/api/errorpb"
	"Airfone/internal/conf"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData)

// Data .
type Data struct {
	// TODO wrapped database client
	idMaker   int32
	consumers *ConsumerMap // 消费者列表
	providers *TopicMap    // 生产者列表
	tasks     chan *Task   // 异步任务管道(处理延时任务)
	log       *log.Helper
}

// 向内传入一个延时任务
func (data *Data) Schedule(task *Task) {
	data.tasks <- task
}

// 获取 ID
//
//	初次连接时用于创建唯一标识 ID
func (data *Data) GetID() int32 {
	return atomic.AddInt32(&data.idMaker, 1)
}

// 添加一个 provider
//
//	思路: 先去拿 topic，拿不到则创建 topic
//	之后将 provider 塞入 topic，注意锁的使用
func (data *Data) AddProvider(topicName string, now int64, id int32, provider *Provider) error {
	t, err := data.providers.GetXTopic(topicName)
	if err != nil {
		return err
	}
	return t.AddProvider(now, id, provider)
}

// 移除一个 provider
//
//	思路：先去拿 topic，拿不到则 报错
//	之后从 topic 中查询 id 在 pending 还是 running 队列
//	若都不存在则报错
func (data *Data) RemoveProvider(topicName string, now int64, id int32) error {
	t, err := data.providers.GetTopic(topicName)
	if err != nil {
		return err
	}
	if _, err = t.GetRunningProvider(id); err == nil {
		t.RemoveRunningProvider(now, id)
		return nil
	}
	if _, err = t.GetPendingProvider(id); err == nil {
		t.RemovePendingProvider(id)
		return nil
	}
	return errorpb.ErrorDeleteInvalid("this provider is not exist, remove faild")
}

// 更新一个 provider
//
//	思路：获取 topic，未获取到报错
//	更新 provider，出错之后先尝试从 pending 队列中将其复活，出错则返回
//	再次尝试更新，若出错则返回
func (data *Data) UpdateProvider(topicName string, now int64, id int32, provider *Provider) error {
	t, err := data.providers.GetTopic(topicName)
	if err != nil {
		return err
	}
	if err = t.UpdateRunningProvider(now, id, provider); err != nil {
		if err = t.Resurrect(now, id); err != nil {
			return err
		}
		return t.UpdateRunningProvider(now, id, provider)
	}
	return nil
}

// 添加一个 consumer
//
//	添加一个消费者
func (data *Data) AddConsumer(now int64, id int32, rely *Rely) error {
	return data.consumers.Add(now, id, rely)
}

// 移除一个 consumer
//
//	移除一个消费者
func (data *Data) RemoveConsumer(id int32) error {
	data.consumers.Remove(id)
	return nil
}

// 更新 consumer 依赖项
//
//	也就是说 consumer 依赖的 topic 有改变
//	比如从依赖 (A,B) 变成 (A,C) 等
func (data *Data) UpdateConsumerRely(now int64, id int32, topics []string) error {
	return data.consumers.Update(now, id, topics)
}

// 更新 consumer 依赖状况
//
//	这是 consumer 依赖的 topic 未变
//	但是被依赖的 provider 自身出现变动应该触发的函数
//	比如 provider current 变化，或者 provider 失联，被
//	移动到 pending 队列中等情况，在心跳时被消费者检测到
func (data *Data) UpdateConsumerTime(now int64, id int32) error {
	return data.consumers.UpdateTime(now, id)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	var (
		data    *Data                 // data
		endSign = make(chan struct{}) // 用于控制异步调度任务的关闭
	)
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		close(endSign)
	}
	data = &Data{
		consumers: &ConsumerMap{consumers: make(map[int32]*Rely)},
		providers: &TopicMap{topics: make(map[string]*Topic)},
		tasks:     make(chan *Task, 8),
		log:       log.NewHelper(logger),
	}
	// 开启异步延时任务
	go delay_task_hanlder(data, endSign)
	// 开启异步定时任务
	go scheduled_task_handler(data, endSign)
	return data, cleanup, nil
}

// 延时任务处理协程
//
//	每一个方法调用在内部自行处理错误，否则日志难以记录，错误处理也会变得麻烦
//	职责:
//	 1. 当心跳来检测依赖项时，发现某些依赖已超时，启用调度器将失效 provider 放到 pending 队列中
//	 2. todo
func delay_task_hanlder(data *Data, end chan struct{}) {
	for {
		select {
		case task := <-data.tasks:
			switch task.Action {
			case TURN_TO_PENDING:
				turn_to_pending(data, task)
			}
		case <-end:
			return
		}
	}
}

// 定时任务处理协程(3s/per)
//  1. pending 队列中，若 provider 超过了等待时长，将其删除
func scheduled_task_handler(data *Data, end chan struct{}) {
	var (
		ticker = time.NewTicker(3 * time.Second)
	)
	for {
		select {
		case <-ticker.C:
			drop_after_pending(data)
		case <-end:
			return
		}
	}
}
