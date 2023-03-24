package repo

import (
	"Airfone/internal/biz/irepo"
	"Airfone/internal/engine"
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type registerRepo struct {
	data *engine.Data
	log  *log.Helper
}

// NewRegisterRepo .
func NewRegisterRepo(data *engine.Data, logger *log.Helper) irepo.RegisterRepo {
	return &registerRepo{
		data: data,
		log:  logger,
	}
}

// 服务注册
func (repo *registerRepo) Register(ctx context.Context, now int64, service *irepo.Service) (*irepo.Service, error) {
	if s, err := repo.data.AddService(service.Topic, now, service.Service); err != nil {
		return nil, err
	} else {
		service.Service = s
		return service, nil
	}
}

// 服务更新
func (repo *registerRepo) Update(ctx context.Context, now int64, service *irepo.Service) (*irepo.Service, error) {
	if s, err := repo.data.UpdateService(service.Topic, now, service.Service); err != nil {
		return nil, err
	} else {
		service.Service = s
		return service, nil
	}

}

// 服务注销
func (repo *registerRepo) Logout(ctx context.Context, now int64, service *irepo.Service) error {
	_, err := repo.data.RemoveService(service.Topic, now, service.ID)
	return err
}

// 服务发现
func (repo *registerRepo) Discover(ctx context.Context, now int64, service *irepo.Service, relies []string) (*irepo.Service, error) {
	if s, err := repo.data.Discover(now, service.Service, relies); err != nil {
		return nil, err
	} else {
		service.Service = s
		return service, nil
	}
}
