package main

import (
	"context"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
)

type UserRepository struct {
	*lorm.Repository[*User]
}

func NewUserRepository(engine *lorm.Engine) *UserRepository {
	return &UserRepository{
		Repository: engine.Repository[*User](),
	}
}

func (r *UserRepository) ListAdults(ctx context.Context, minAge int) ([]*User, error) {
	var u User
	return r.Engine.Query[*User]().
		Where(builder.Gte(u.LormCols().Age(), minAge)).
		OrderBy(u.LormCols().ID() + " ASC").
		Find(ctx)
}
