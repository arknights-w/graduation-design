package repo

import (
	"Airfone/internal/biz/irepo"
	"Airfone/internal/engine"
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type keepAliveRepo struct {
	data *engine.Data
	log  *log.Helper
}

// NewkeepAliveRepoRepo .
func NewkeepAliveRepoRepo(data *engine.Data, logger *log.Helper) irepo.KeepAliveRepo {
	return &keepAliveRepo{
		data: data,
		log:  logger,
	}
}

func (repo *keepAliveRepo) KeepAlive(ctx context.Context, now int64, hb *irepo.HeartBeat) (*irepo.HeartBeat, error) {
	hb2, err := repo.data.Check(now, hb.HeartBeat)
	if err != nil {
		return nil, err
	}
	hb.HeartBeat = hb2
	return hb, nil
}

func (repo *keepAliveRepo) Conform(ctx context.Context, now int64, hb *irepo.HeartBeat) error {
	return repo.data.Conform(now, hb.Topic, hb.ID)
}
