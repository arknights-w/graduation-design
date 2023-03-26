package service

import (
	"context"

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
		service  = &irepo.Service{}
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
	return response, nil
}

func (s *AirfoneService) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	var (
		service  = &irepo.Service{}
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
	if req.NeedRelies{
		if len(req.Relies) == 0{
			service, err = s.ruc.Update(ctx, service, make([]string, 0))
		}else {
			service, err = s.ruc.Update(ctx, service, req.Relies)
		}
	}else {
		service, err = s.ruc.Update(ctx, service, nil)
	}
	if err != nil {
		return nil, err
	}

	response.Service = service.ToProto()
	return response, nil
}

func (s *AirfoneService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	var (
		serv = &irepo.Service{}
		err  error
	)
	serv.Topic = req.Topic
	serv.ID = req.Id
	if err = s.ruc.Logout(ctx, serv); err != nil {
		return nil, err
	}
	return &pb.LogoutResponse{}, nil
}

func (s *AirfoneService) KeepAlive(ctx context.Context, req *pb.KeepAliveRequest) (*pb.KeepAliveResponse, error) {
	var (
		hb  = &irepo.HeartBeat{}
		res = &pb.KeepAliveResponse{}
		err error
	)
	hb.ID = req.Id
	hb.Topic = req.Topic

	if hb, err = s.kuc.KeepAlive(ctx, hb); err != nil {
		return nil, err
	}
	res.Keepalive = hb.ToProto()
	return res, nil
}

func (s *AirfoneService) Conform(ctx context.Context, req *pb.ConformRequest) (*pb.ConformResponse, error) {
	var (
		hb = &irepo.HeartBeat{}
		err error
	)
	hb.ID = req.Id
	hb.Topic = req.Topic
	if err = s.kuc.Conform(ctx, hb); err != nil {
		return nil,err
	}
	return &pb.ConformResponse{},nil
}
