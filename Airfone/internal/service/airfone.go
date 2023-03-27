package service

import (
	"context"
	"fmt"

	pb "Airfone/api/airfone"
	"Airfone/internal/biz"
	"Airfone/internal/biz/irepo"
	"Airfone/internal/engine"
)

type AirfoneService struct {
	pb.UnimplementedAirfoneServer
	kuc *biz.KeepAliveUsecase
	ruc *biz.RegisterUsecase
}

func NewAirfoneService(kuc *biz.KeepAliveUsecase, ruc *biz.RegisterUsecase) *AirfoneService {
	return &AirfoneService{
		kuc: kuc,
		ruc: ruc,
	}
}

func (s *AirfoneService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	var (
		service = &irepo.Service{
			Service: &engine.Service{},
		}
		relies   = req.Relies
		schema   = make([]*engine.Schema, len(req.Schema))
		response = &pb.RegisterResponse{}
	)
	for i, s2 := range req.Schema {
		schema[i] = &engine.Schema{
			Title:   s2.Title,
			Content: s2.Content,
		}
	}
	service.Topic = req.Topic
	service.IP = req.Ip
	service.Port = uint16(req.Port)
	service.Schema = schema

	s2, err := s.ruc.Register(ctx, service, relies)
	if err != nil {
		return nil, err
	}

	response.Service = s2.ToProto()
	fmt.Printf("服务 %v id: %v 注册成功, %v:%v   \n", s2.Topic, s2.ID, s2.IP, s2.Port)
	return response, nil
}

func (s *AirfoneService) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	var (
		service = &irepo.Service{
			Service: &engine.Service{},
		}
		schema   = make([]*engine.Schema, len(req.Schema))
		response = &pb.UpdateResponse{}
		err      error
	)
	service.ID = req.Id
	service.Topic = req.Topic
	service.IP = req.Ip
	service.Port = uint16(req.Port)
	if req.NeedSchema {
		for i, s2 := range req.Schema {
			schema[i] = &engine.Schema{
				Title:   s2.Title,
				Content: s2.Content,
			}
		}
		service.Schema = schema
	}
	if req.NeedRelies {
		if len(req.Relies) == 0 {
			service, err = s.ruc.Update(ctx, service, make([]string, 0))
		} else {
			service, err = s.ruc.Update(ctx, service, req.Relies)
		}
	} else {
		service, err = s.ruc.Update(ctx, service, nil)
	}
	if err != nil {
		return nil, err
	}

	response.Service = service.ToProto()
	fmt.Printf("服务 %v id: %v 更新成功, %v:%v   \n", service.Topic, service.ID, service.IP, service.Port)
	return response, nil
}

func (s *AirfoneService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	var (
		serv = &irepo.Service{
			Service: &engine.Service{},
		}
		err error
	)
	serv.Topic = req.Topic
	serv.ID = req.Id
	if err = s.ruc.Logout(ctx, serv); err != nil {
		return nil, err
	}
	fmt.Printf("服务 %v id: %v 注销成功\n", serv.Topic, serv.ID)
	return &pb.LogoutResponse{}, nil
}

func (s *AirfoneService) KeepAlive(ctx context.Context, req *pb.KeepAliveRequest) (*pb.KeepAliveResponse, error) {
	var (
		hb = &irepo.HeartBeat{
			HeartBeat: &engine.HeartBeat{},
		}
		res = &pb.KeepAliveResponse{}
		err error
	)
	hb.ID = req.Id
	hb.Topic = req.Topic

	if hb, err = s.kuc.KeepAlive(ctx, hb); err != nil {
		return nil, err
	}
	res.Keepalive = hb.ToProto()
	fmt.Printf("服务 %v id: %v 心跳检测\n", hb.Topic, hb.ID)
	return res, nil
}

func (s *AirfoneService) Conform(ctx context.Context, req *pb.ConformRequest) (*pb.ConformResponse, error) {
	var (
		hb = &irepo.HeartBeat{
			HeartBeat: &engine.HeartBeat{},
		}
		err error
	)
	hb.ID = req.Id
	hb.Topic = req.Topic
	if err = s.kuc.Conform(ctx, hb); err != nil {
		return nil, err
	}
	fmt.Printf("服务 %v id: %v 服务运行确认\n", hb.Topic, hb.ID)
	return &pb.ConformResponse{}, nil
}
