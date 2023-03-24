package biz

import (
	"Airfone/internal/biz/irepo"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type RegisterUsecase struct {
	repo irepo.RegisterRepo
	log  *log.Helper
}

func NewRegisterUsecase(repo irepo.RegisterRepo, logger *log.Helper) *RegisterUsecase {
	return &RegisterUsecase{
		repo: repo,
		log:  logger,
	}
}

// 服务注册
//
//	思路: 先进行服务发现，找到他所有的依赖，然后再调用 repo 层的注册方法完成注册
func (uc *RegisterUsecase) Register(ctx context.Context, serv *irepo.Service, relies []string) (*irepo.Service, error) {
	var (
		now = time.Now()
		err error
	)
	if serv, err = uc.repo.Discover(ctx, now.UnixNano(), serv, relies); err != nil {
		return nil, err
	}
	return uc.repo.Register(ctx, now.UnixNano(), serv)
}

// 服务更新
//
//	思路: 先看是否有依赖更新，有则先进行服务发现
//	再调用 repo update 函数进行更新
func (uc *RegisterUsecase) Update(ctx context.Context, serv *irepo.Service, relies []string) (*irepo.Service, error) {
	var (
		now = time.Now()
		err error
	)
	if relies != nil {
		if serv, err = uc.repo.Discover(ctx, now.UnixNano(), serv, relies); err != nil {
			return nil, err
		}
	}
	return uc.repo.Update(ctx, now.UnixNano(), serv)
}

func (uc *RegisterUsecase) Logout(ctx context.Context, serv *irepo.Service) error {
	var (
		now = time.Now()
	)
	return uc.repo.Logout(ctx, now.UnixNano(), serv)
}
