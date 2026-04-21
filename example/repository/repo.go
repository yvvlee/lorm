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
		Repository: lorm.NewRepository[*User](engine),
	}
}

func (r *UserRepository) ListAdults(ctx context.Context, minAge int) ([]*User, error) {
	var u User
	return lorm.Query[*User](r.Engine).
		Where(builder.Gte(u.Fields().Age(), minAge)).
		OrderBy(u.Fields().ID() + " ASC").
		Find(ctx)
}
