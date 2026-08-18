package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestRepository interface {
	Get(ctx context.Context, id any) (*Test, error)
	GetByField(ctx context.Context, field string, value any) (*Test, error)
	Lock(ctx context.Context, id any) (*Test, error)
	LockByField(ctx context.Context, field string, value any) (*Test, error)
	Exist(ctx context.Context, id any) (bool, error)
	ExistByField(ctx context.Context, field string, value any) (bool, error)
	Update(ctx context.Context, user *Test) (rowsAffected int64, err error)
	UpdateMap(ctx context.Context, id any, data map[string]any) (rowsAffected int64, err error)
	Insert(ctx context.Context, user *Test) (rowsAffected int64, err error)
	InsertAll(ctx context.Context, users []*Test) (rowsAffected int64, err error)
	Delete(ctx context.Context, id any) (rowsAffected int64, err error)
	DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error)
}

var _ TestRepository = (*TestRepositoryImpl)(nil)

type TestRepositoryImpl struct {
	*Repository[*Test]
}

func NewTestRepository(engine *Engine) *TestRepositoryImpl {
	return &TestRepositoryImpl{
		Repository: engine.Repository[*Test](),
	}
}

func TestRepositoryWrapperPrimaryKeyErrors(t *testing.T) {
	engine := &Engine{config: &Config{}}

	noPKRepo := engine.Repository[*testNoPrimaryKeyModel]()
	_, err := noPKRepo.Lock(context.Background(), 1)
	assert.ErrorContains(t, err, "single-column primary keys")
	_, err = noPKRepo.Exist(context.Background(), 1)
	assert.ErrorContains(t, err, "single-column primary keys")

	compositeRepo := engine.Repository[*testCompositePrimaryKeyModel]()
	_, err = compositeRepo.Lock(context.Background(), 1)
	assert.ErrorContains(t, err, "single-column primary keys")
	_, err = compositeRepo.Exist(context.Background(), 1)
	assert.ErrorContains(t, err, "single-column primary keys")
}
