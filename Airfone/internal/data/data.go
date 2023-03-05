package data

import (
	"Airfone/internal/conf"
	"Airfone/internal/data/engine"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewGreeterRepo)

// Data .
type Data struct {
	// TODO wrapped database client
	consumers *engine.ConsumerMap   // 消费者列表
	providers *engine.TopicMap      // 生产者列表
	Schedules chan *engine.Schedule // 异步任务管道
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	var (
		data    *Data                 // data
		endSign = make(chan struct{}) // 用于控制异步调度任务的关闭
	)
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		// close(endSign)
	}
	data = &Data{
		consumers: new(engine.ConsumerMap),
		providers: new(engine.TopicMap),
		Schedules: make(chan *engine.Schedule, 10),
	}
	return data, cleanup, nil
}

// 一个协程异步处理数据状态变化
// 职责:
//  1. 当心跳来检测依赖项时，发现某些依赖已超时，启用调度器将失效 provider 放到 pending 队列中
//  2. pending 队列中，若 provider 超过了等待时长，将其删除
//  3. todo
func (data *Data) scheduler(end chan struct{}) {
LOOP:
	for {
		select {
		case schedule := <-data.Schedules:
			switch schedule.Action {
			case engine.TURN_TO_PENDING:
				// todo
			case engine.DELETE_FROM_PENDING:
				//todo
			}
		case <-end:
			break LOOP
		}
	}
}
