package cli

import (
	pb "cli/api/airfone"
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DURATION_HEARTBEAT = 2 * time.Second // 心跳时间，每隔 DURATION_HEARTBEAT 轮询一次，客户端向服务器端发送一次心跳
	DURATION_VALID     = 3 * time.Second // 有效时间，当客户端超过 DURATION_VALID 未向服务器端发送心跳，即便在 running 队列中，也不纳入使用
	DURATION_PENDING   = 4 * time.Second // pending时间，若 service心跳 与当前时间相差 DURATION_PENDING 以上，则将其从running队列放到pending队列
	DURATION_DROPPED   = 6 * time.Second // dropping时间，若 pending队列中 service心跳 与当前时间相差 DURATION_DROPPED 则认为失联，将其状态置为 dropping 并删除
)

type Config struct {
	Schema []*pb.Schema // 元数据
	Relies []string     // 依赖的 topic 名
	Topic  string       // 自己的主题
	IP     string       // ip
	Port   int32        // 端口
}

type client struct {
	proto  pb.AirfoneClient
	conn   *grpc.ClientConn
	ctx    context.Context
	cancel chan struct{}

	Relies map[string]*pb.Rely // 依赖
	Schema []*pb.Schema        // 元数据信息
	Topic  string              // 服务名称
	Ip     string              // ip地址
	Prot   int32               // 端口
	Id     int32               // id 号
	Status pb.HeartBeatType    // 服务状态
}

// 新建一个客户端
//
//	url 为注册中心的 ip + port
func NewCli(url string) (*client, error) {
	var (
		client   = new(client)
		security grpc.DialOption
		conn     *grpc.ClientConn
		err      error
	)
	security = grpc.WithTransportCredentials(insecure.NewCredentials())
	if conn, err = grpc.Dial(url, security); err != nil {
		return nil, err
	}

	cli := pb.NewAirfoneClient(conn)
	client.conn = conn
	client.proto = cli
	return client, nil
}

// 注册
func (cli *client) Register(cfg *Config) error {
	var (
		ctx, cancelFunc = context.WithCancel(context.Background())
		cancel          = make(chan struct{})
	)
	cli.cancel = cancel
	cli.ctx = ctx

	// 进行服务注册
	res, err := cli.proto.Register(ctx, &pb.RegisterRequest{
		Schema: cfg.Schema,
		Relies: cfg.Relies,
		Topic:  cfg.Topic,
		Ip:     cfg.IP,
		Port:   cfg.Port,
	})
	if err != nil {
		cancelFunc()
		return err
	}
	cli.fromService(res.Service)

	// 如果状态为 changed 则需要更新依赖服务，重新向服务端发送确认信息，确保服务可用
	if cli.Status == pb.HeartBeatType_HeartBeat_CHANGED {
		if _, err = cli.proto.Conform(ctx, &pb.ConformRequest{
			Topic: cli.Ip,
			Id:    cli.Id,
		}); err != nil {
			cancelFunc()
			return nil
		}
		cli.Status = pb.HeartBeatType_HeartBeat_RUNNING
	}

	go cli.keepalive(cancelFunc)

	return nil
}

// 心跳
//
//	被动的，在注册的时候自动启动，注销的时候自动删除
func (cli *client) keepalive(cancel func()) {
	var (
		ticker = time.NewTicker(DURATION_HEARTBEAT)
	)
	for {
		select {
		case <-ticker.C:
			// 心跳检测
			res, err := cli.proto.KeepAlive(cli.ctx, &pb.KeepAliveRequest{
				Topic: cli.Topic,
				Id:    cli.Id,
			})
			if err != nil {
				fmt.Println(err)
				continue
			}
			if res.Keepalive.Status == pb.HeartBeatType_HeartBeat_CHANGED {
				// 若心跳状态为 changed, 则需要再次确认
				cli.updateRelies(res.Keepalive.Relies)
				_, err := cli.proto.Conform(cli.ctx, &pb.ConformRequest{
					Topic: cli.Topic,
					Id:    cli.Id,
				})
				if err != nil {
					fmt.Println(err)
					continue
				}
				cli.Status = pb.HeartBeatType_HeartBeat_RUNNING
			} else if res.Keepalive.Status == pb.HeartBeatType_HeartBeat_DROPPED {
				// todo
				// 若心跳状态为 dropped, 则需要重新注册
				var (
					relies = make([]string, 0, len(cli.Relies))
					res    *pb.RegisterResponse
				)
				for _, r := range cli.Relies {
					relies = append(relies, r.Topic)
				}
				if res, err = cli.proto.Register(cli.ctx, &pb.RegisterRequest{
					Schema: cli.Schema,
					Relies: relies,
					Topic:  cli.Topic,
					Ip:     cli.Ip,
					Port:   cli.Prot,
				}); err != nil {
					fmt.Println(err)
					continue
				}
				cli.fromService(res.Service)
				// 如果状态为 changed 则需要更新依赖服务，重新向服务端发送确认信息，确保服务可用
				if cli.Status == pb.HeartBeatType_HeartBeat_CHANGED {
					if _, err = cli.proto.Conform(cli.ctx, &pb.ConformRequest{
						Topic: cli.Ip,
						Id:    cli.Id,
					}); err != nil {
						fmt.Println(err)
						continue
					}
					cli.Status = pb.HeartBeatType_HeartBeat_RUNNING
				}
			}
		case <-cli.cancel:
			cancel()
			return
		}
	}
}

// 只需要填写需要修改的配置项
//
//	当需要将元数据或依赖从 n 个修改为 0 个
//	传入 make([]string,0) 或 make([]*pb.Schema,0)
type UpdateConfig struct {
	Schema []*pb.Schema // 元数据
	Relies []string     // 依赖的 topic 名
	IP     string       // ip
	Port   int32        // 端口
}

// 主动更新
func (cli *client) Update(cfg *UpdateConfig) error {
	var (
		req = &pb.UpdateRequest{}
		res = &pb.UpdateResponse{}
		err error
	)
	req.Id = cli.Id
	req.Topic = cli.Topic
	if cfg.IP != "" {
		req.Ip = cfg.IP
	}
	if cfg.Port != 0 {
		req.Port = cfg.Port
	}
	if cfg.Relies != nil {
		req.NeedRelies = true
		req.Relies = cfg.Relies
	}
	if cfg.Schema != nil {
		req.NeedSchema = true
		req.Schema = cfg.Schema
	}
	if res, err = cli.proto.Update(cli.ctx, req); err != nil {
		return err
	}
	cli.fromService(res.Service)
	switch cli.Status {
	case pb.HeartBeatType_HeartBeat_PENDING:
		// todo
	case pb.HeartBeatType_HeartBeat_RUNNING:
		// todo
	case pb.HeartBeatType_HeartBeat_CHANGED:
		// todo
		if _, err = cli.proto.Conform(cli.ctx, &pb.ConformRequest{
			Topic: cli.Topic,
			Id:    cli.Id,
		}); err != nil {
			return err
		}
	}
	return nil
}

// 注销
func (cli *client) Logout() error {
	if _, err := cli.proto.Logout(cli.ctx, &pb.LogoutRequest{
		Topic: cli.Topic,
		Id:    cli.Id,
	}); err != nil {
		return err
	}
	cli.cancel <- struct{}{}
	cli.ctx.Done()
	if err := cli.conn.Close(); err != nil {
		return err
	}
	return nil
}

// 从 service 中获取
func (cli *client) fromService(serv *pb.Service) {
	cli.Id = serv.Id
	cli.Ip = serv.Ip
	cli.Prot = serv.Prot
	cli.Topic = serv.Topic
	cli.Status = serv.Status
	cli.Schema = serv.Schema
	cli.setRelies(serv.Relies)
}

// 更新 Relies
//
//	以前存在的依赖不会受到印象
func (cli *client) updateRelies(rely []*pb.Rely) {
	for _, r := range rely {
		cli.Relies[r.Topic] = r
	}
}

// 设置 Relies
//
//	清空以前全部的依赖，重新获取当前切片中的依赖
func (cli *client) setRelies(rely []*pb.Rely) {
	var (
		newRelies = make(map[string]*pb.Rely)
	)
	for _, r := range rely {
		newRelies[r.Topic] = r
	}
	cli.Relies = newRelies
}
