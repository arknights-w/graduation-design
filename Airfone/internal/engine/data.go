package engine

import (
	"Airfone/api/errorpb"
	"Airfone/internal/conf"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is data services.
var ProviderSet = wire.NewSet(NewData, NewHelper)

func NewHelper(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}

// Data .
type Data struct {
	// TODO wrapped database client
	sync.RWMutex
	idMaker int32
	topics  map[string]*Topic // 生产者列表
	log     *log.Helper
}

// 获取 ID
//
//	初次连接时用于创建唯一标识 ID
func (data *Data) getID() int32 {
	return atomic.AddInt32(&data.idMaker, 1)
}

// 添加一个 service
//
//	思路: 先去拿 topic，拿不到则创建 topic
//	之后将 service 塞入 topic，注意锁的使用
func (data *Data) AddService(topicName string, now int64, service *Service) (*Service, error) {
	t, err := data.getXTopic(topicName)
	if err != nil {
		return nil, err
	}
	service.ID = data.getID()
	switch service.Status {
	default:
		return t.AddRunningService(now, service)
	case HeartBeat_PENDING:
		return t.AddPendingService(now, service)
	case HeartBeat_DROPPED:
		return nil, errorpb.ErrorInsertAlreadyExist("service has been dropped status")
	}
}

// 移除一个 service
//
//	思路：先去拿 topic，拿不到则 报错
//	之后从 topic 中查询 id 在 pending 还是 running 队列
//	若都不存在则报错
func (data *Data) RemoveService(topicName string, now int64, id int32) (*Service, error) {
	t, err := data.getTopic(topicName)
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
//	这个方法可以修改除 id, topic 以外的其他全部属性
//	思路：获取 topic，未获取到报错
//	更新 service，出错之后先尝试从 pending 队列中将其复活，出错则返回
//	复活成功后，再次尝试更新，若出错则返回
func (data *Data) UpdateService(topicName string, now int64, serv *Service) (*Service, error) {
	var (
		topic   *Topic
		service *Service
		err     error
	)
	topic, err = data.getTopic(topicName)
	if err != nil {
		return nil, err
	}
	if service, err = topic.UpdateRunningService(now, serv); err != nil {
		if _, err = topic.Resurrect(now, serv.ID); err != nil {
			return nil, err
		}
		return topic.UpdateRunningService(now, serv)
	}
	return service, nil
}

// 内部使用的 topic 类型，不对外暴露
type innerTopic struct {
	topicName string
	*Topic
}

// 服务发现
//
//	这个方法可能修改服务的 status，rely 两个属性
//	思路: data上读锁, 挨个读取所有依赖的topic
//	running列表上读锁，再从 topic 中随机选取一个 running 节点
//	如果该 running 中为空，则将(discover方法传入的)服务 status 置为 pending
//	将所有的 rely 塞入 service, 然后返回
func (data *Data) Discover(now int64, service *Service, relies []string) (*Service, error) {
	var (
		topics = make([]*innerTopic, 0, len(relies))
		status = HeartBeat_RUNNING
		rely   []*Rely
	)
	// 不需要依赖的话直接跳过
	if len(relies) == 0 {
		service.Rely = rely
		service.Status = status
		return service, nil
	}

	// 读取所有 topic
	data.RLock()
	for _, t := range relies {
		topics = append(topics, &innerTopic{
			Topic:     data.topics[t],
			topicName: t,
		})
	}
	data.RUnlock()

	rely, status = data.discover(now, topics)

	// rely 塞入 service，返回
	service.Rely = rely
	service.Status = status
	return service, nil
}

// 心跳检查
//
//	思路：先获取 topic,再获取 service，
//	若未获取到 service 则认为 service 因延迟被删除，需要重新注册
//	再通过 service rely 来检测是否有依赖出故障
//	最终还需要修改服务状态:
//	若当前状态为 running, changed 则放置于 running 队列中
//	若当前状态为 pending 则放置于 pending 队列中
func (data *Data) Check(now int64, hb *HeartBeat) (*HeartBeat, error) {
	var (
		status    = HeartBeat_RUNNING // 心跳状态
		repyTopic []*innerTopic       // 依赖的主题
		relies    []*Rely             // 需要改变的依赖
		topic     *Topic              // 心跳的主题
		serv      *Service            // 心跳的service
		err       error
	)
	// 获取 topic
	if topic, err = data.getTopic(hb.Topic); err != nil {
		return nil, err
	}
	// 获取 service
	if serv, err = topic.GetService(hb.ID); err != nil {
		status = HeartBeat_DROPPED
		hb.Status = status
		return hb, nil
	}
	// 获取到serv 后将其更新时间置为当前时间
	serv.keepalive = now

	// 探测过期的依赖
	// 若他原本没有依赖，则直接返回
	if len(serv.Rely) != 0 {
		repyTopic = make([]*innerTopic, 0, len(serv.Rely))
		data.RLock()
		limit := now - int64(DURATION_PENDING)
		for _, r := range serv.Rely {
			if *r.Keepalive < limit || *r.Status != HeartBeat_RUNNING {
				repyTopic = append(repyTopic, &innerTopic{
					Topic:     data.topics[r.Topic],
					topicName: r.Topic,
				})
			}
		}
		data.RUnlock()

		// 若所有依赖均正常,状态为running，则直接返回
		// 若依赖有故障则修改状态为changed，并进行一次服务发现
		// 若服务发现后 serv 状态为pending，则状态修改为 pending
		if len(repyTopic) != 0 {
			relies, status = data.discover(now, repyTopic)
		}
	}

	// 赋值返回
	hb.Rely = relies
	hb.Status = status

	// 服务状态变更
	switch status {
	case HeartBeat_RUNNING, HeartBeat_CHANGED:
		serv.Status = status
		topic.ResurrectX(now, hb.ID)
	case HeartBeat_PENDING:
		serv.Status = status
		topic.PendX(now, hb.ID)
	}

	return hb, nil
}

func (data *Data) Conform(now int64, topicName string, id int32) error {
	var (
		topic *Topic
		err   error
	)
	if topic, err = data.getTopic(topicName); err != nil {
		return err
	}
	return topic.Conform(now, id)
}

// 内部服务发现
//
//	最终返回的状态可能是
//	当所有主题都能正常找到新依赖时，返回changed
//	当有主题中没有可用依赖时，返回pending
func (data *Data) discover(now int64, topics []*innerTopic) ([]*Rely, HeartBeatType) {
	var (
		rely   = make([]*Rely, 0, len(topics))
		status = HeartBeat_CHANGED
	)
	// 遍历 topic，取出其中心跳正常的 service
	for _, t := range topics {
		var (
			list []*Service
			serv *Service
		)
		list = t.GetAllRunningService(now)
		// 如果该 topic 没有 service 在正常心跳范围
		// 则将当前传入的 service 状态置为 pending
		if len(list) > 0 {
			serv = list[rand.Intn(len(list))]
			rely = append(rely, &Rely{
				Keepalive: &serv.keepalive,
				Status:    &serv.Status,
				Topic:     t.topicName,
				IP:        serv.IP,
				ID:        serv.ID,
				Port:      serv.Port,
			})
		} else {
			status = HeartBeat_PENDING
		}
	}
	return rely, status
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
		topics: make(map[string]*Topic),
		log:    logger,
	}
	return data, cleanup, nil
}
