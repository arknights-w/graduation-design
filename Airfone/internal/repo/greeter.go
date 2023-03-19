package repo

import (
	"context"

	"Airfone/internal/biz/irepo"
	"Airfone/internal/engine"

	"github.com/go-kratos/kratos/v2/log"
)

type greeterRepo struct {
	data *engine.Data
	log  *log.Helper
}

// NewGreeterRepo .
func NewGreeterRepo(data *engine.Data, logger *log.Helper) irepo.GreeterRepo {
	return &greeterRepo{
		data: data,
		log:  logger,
	}
}

func (r *greeterRepo) Save(ctx context.Context, g *irepo.Greeter) (*irepo.Greeter, error) {
	return g, nil
}

func (r *greeterRepo) Update(ctx context.Context, g *irepo.Greeter) (*irepo.Greeter, error) {
	return g, nil
}

func (r *greeterRepo) FindByID(context.Context, int64) (*irepo.Greeter, error) {
	return nil, nil
}

func (r *greeterRepo) ListByHello(context.Context, string) ([]*irepo.Greeter, error) {
	return nil, nil
}

func (r *greeterRepo) ListAll(context.Context) ([]*irepo.Greeter, error) {
	return nil, nil
}
