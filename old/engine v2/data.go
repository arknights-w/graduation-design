package engine

import (
	"Airfone/api/errorpb"
	"Airfone/internal/conf"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data services.
var ProviderSet = wire.NewSet(NewData, NewHelper)

// Data .
type Data struct {
	// TODO wrapped database client
	idMaker  int32
	services *TopicMap // 生产者列表
	log      *log.Helper
}

// 获取 ID
//
//	初次连接时用于创建唯一标识 ID
func (data *Data) GetID() int32 {
	return atomic.AddInt32(&data.idMaker, 1)
}

// 添加一个 service
//
//	思路: 先去拿 topic，拿不到则创建 topic
//	之后将 service 塞入 topic，注意锁的使用
func (data *Data) AddService(topicName string, now int64, id int32, service *Service) error {
	t, err := data.services.GetXTopic(topicName)
	if err != nil {
		return err
	}
	return t.AddService(now, id, service)
}

// 移除一个 service
//
//	思路：先去拿 topic，拿不到则 报错
//	之后从 topic 中查询 id 在 pending 还是 running 队列
//	若都不存在则报错
func (data *Data) RemoveService(topicName string, now int64, id int32) (*Service, error) {
	t, err := data.services.GetTopic(topicName)
	if err != nil {
		return nil, err
	}
	if _, err = t.GetRunningService(id); err == nil {
		return t.RemoveRunningService(now, id)
	}
	if _, err = t.GetPendingService(id); err == nil {
		return t.RemovePendingService(now, id)

	}
	return nil, errorpb.ErrorDeleteInvalid("this service is not exist, remove faild")
}

// 更新一个 service
//
//	思路：获取 topic，未获取到报错
//	更新 service，出错之后先尝试从 pending 队列中将其复活，出错则返回
//	复活成功后，再次尝试更新，若出错则返回
func (data *Data) UpdateService(topicName string, now int64, id int32, serv *Service) (*Service, error) {
	var (
		topic   *Topic
		service *Service
		err     error
	)
	topic, err = data.services.GetTopic(topicName)
	if err != nil {
		return nil, err
	}
	if service, err = topic.UpdateRunningService(now, id, serv); err != nil {
		if err = topic.Resurrect(now, id); err != nil {
			return nil, err
		}
		return topic.UpdateRunningService(now, id, serv)
	}
	return service, nil
}

// NewData .
func NewData(c *conf.Data, logger *log.Helper) (*Data, func(), error) {
	var (
		data    *Data                 // data
		endSign = make(chan struct{}) // 用于控制异步调度任务的关闭
	)
	cleanup := func() {
		log.Info("closing the data resources")
		close(endSign)
	}
	data = &Data{
		services: NewTopicMap(logger),
		log:      logger,
	}
	return data, cleanup, nil
}

func NewHelper(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}
