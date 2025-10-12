package lorm

import (
	"context"
)

type TestRepository interface {
	Get(ctx context.Context, id int64) (*Test, error)
	GetByField(ctx context.Context, field string, value any) (*Test, error)
	Lock(ctx context.Context, id int64) (*Test, error)
	LockByField(ctx context.Context, field string, value any) (*Test, error)
	Exist(ctx context.Context, id int64) (bool, error)
	ExistByField(ctx context.Context, field string, value any) (bool, error)
	Update(ctx context.Context, user *Test) (rowsAffected int64, err error)
	UpdateMap(ctx context.Context, id int64, data map[string]any) (rowsAffected int64, err error)
	Insert(ctx context.Context, user *Test) (rowsAffected int64, err error)
	InsertAll(ctx context.Context, users []*Test) (rowsAffected int64, err error)
	Delete(ctx context.Context, id int64) (rowsAffected int64, err error)
	DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error)
}

var _ TestRepository = (*TestRepositoryImpl)(nil)

type TestRepositoryImpl struct {
	*Repository[*Test]
}

func NewTestRepository(engine *Engine) *TestRepositoryImpl {
	return &TestRepositoryImpl{
		Repository: NewRepository[*Test](engine),
	}
}
