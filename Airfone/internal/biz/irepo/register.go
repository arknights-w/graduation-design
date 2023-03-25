package irepo

import (
	"context"

	pb "Airfone/api/airfone"
	"Airfone/internal/engine"
)

type Service struct {
	*engine.Service
	Topic string
}

func (s *Service) ToProto() *pb.Service {
	var (
		relies  = make([]*pb.Rely, len(s.Rely))
		schema  = make([]*pb.Schema, len(s.Rely))
		service = &pb.Service{
			Topic:  s.Topic,
			Ip:     s.IP,
			Prot:   s.ID,
			Id:     s.ID,
			Status: statusHeartBeatToProto[s.Status],
		}
	)
	for i, r := range s.Rely {
		relies[i] = &pb.Rely{
			Topic: r.Topic,
			Ip:   r.IP,
			Port:  int32(r.Port),
			Id:    r.ID,
		}
	}
	for i, s2 := range s.Schema {
		schema[i] = &pb.Schema{
			Title:   s2.Title,
			Content: s2.Content,
		}
	}
	service.Relies = relies
	service.Schema = schema
	return service
}

type RegisterRepo interface {
	Register(ctx context.Context, now int64, service *Service) (*Service, error)                  // 服务注册
	Update(ctx context.Context, now int64, service *Service) (*Service, error)                    // 服务更新
	Logout(ctx context.Context, now int64, service *Service) error                                // 服务注销
	Discover(ctx context.Context, now int64, service *Service, relies []string) (*Service, error) // 服务发现
}
