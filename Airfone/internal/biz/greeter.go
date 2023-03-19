package biz

import (
	"context"

	"Airfone/api/errorpb"
	"Airfone/internal/biz/irepo"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	// ErrUserNotFound is user not found.
	ErrUserNotFound = errors.NotFound(errorpb.ErrorReason_USER_NOT_FOUND.String(), "user not found")
)

// GreeterUsecase is a Greeter usecase.
type GreeterUsecase struct {
	repo irepo.GreeterRepo
	log  *log.Helper
}

// NewGreeterUsecase new a Greeter usecase.
func NewGreeterUsecase(repo irepo.GreeterRepo, logger *log.Helper) *GreeterUsecase {
	return &GreeterUsecase{repo: repo, log: logger}
}

// CreateGreeter creates a Greeter, and returns the new Greeter.
func (uc *GreeterUsecase) CreateGreeter(ctx context.Context, g *irepo.Greeter) (*irepo.Greeter, error) {
	uc.log.WithContext(ctx).Infof("CreateGreeter: %v", g.Hello)
	return uc.repo.Save(ctx, g)
}
