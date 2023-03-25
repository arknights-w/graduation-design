package irepo

import (
	pb "Airfone/api/airfone"
	"Airfone/internal/engine"
	"context"
)

var (
	statusHeartBeatToEngine = map[pb.HeartBeatType]engine.HeartBeatType{
		pb.HeartBeatType_HeartBeat_RUNNING: engine.HeartBeat_RUNNING,
		pb.HeartBeatType_HeartBeat_CHANGED: engine.HeartBeat_CHANGED,
		pb.HeartBeatType_HeartBeat_PENDING: engine.HeartBeat_PENDING,
		pb.HeartBeatType_HeartBeat_DROPPED: engine.HeartBeat_DROPPED,
	}

	statusHeartBeatToProto = map[engine.HeartBeatType]pb.HeartBeatType{
		engine.HeartBeat_RUNNING: pb.HeartBeatType_HeartBeat_RUNNING,
		engine.HeartBeat_CHANGED: pb.HeartBeatType_HeartBeat_CHANGED,
		engine.HeartBeat_PENDING: pb.HeartBeatType_HeartBeat_PENDING,
		engine.HeartBeat_DROPPED: pb.HeartBeatType_HeartBeat_DROPPED,
	}
)

// KeepAlive
type HeartBeat struct {
	*engine.HeartBeat
}

func (hb *HeartBeat) ToProto() *pb.Keepalive {
	var (
		relies    = make([]*pb.Rely, len(hb.Rely))
		keepalive = &pb.Keepalive{
			Status: statusHeartBeatToProto[hb.Status],
		}
	)
	for i, r := range hb.Rely {
		relies[i] = &pb.Rely{
			Topic: r.Topic,
			Ip:   r.IP,
			Port:  int32(r.Port),
			Id:    r.ID,
		}
	}
	keepalive.Relies = relies
	return keepalive
}

type KeepAliveRepo interface {
	KeepAlive(ctx context.Context, now int64, hb *HeartBeat) (*HeartBeat, error)
	Conform(ctx context.Context, now int64, hb *HeartBeat) error //在检测到依赖修改后，需要发送 conform 保证自己的服务可用
}
