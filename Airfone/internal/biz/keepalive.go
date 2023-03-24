package biz

import (
	"Airfone/internal/biz/irepo"
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type KeepAliveUsecase struct {
	repo irepo.KeepAliveRepo
	log  *log.Helper
}

func NewKeepAliveUsecase(repo irepo.KeepAliveRepo, logger *log.Helper) *KeepAliveUsecase {
	return &KeepAliveUsecase{
		repo: repo,
		log:  logger,
	}
}

func (uc *KeepAliveUsecase) KeepAlive(ctx context.Context, hb *irepo.HeartBeat) (*irepo.HeartBeat, error) {
	var (
		now = time.Now()
	)
	return uc.repo.KeepAlive(ctx, now.UnixNano(), hb)
}

func (uc *KeepAliveUsecase) Conform(ctx context.Context, hb *irepo.HeartBeat) error {
	var (
		now = time.Now()
	)
	return uc.repo.Conform(ctx, now.UnixNano(), hb)
}
