package lorm

import (
	"context"
	"errors"

	"github.com/yvvlee/lorm/builder"
)

type Repository[T Table] struct {
	Engine *Engine
}

func NewRepository[T Table](engine *Engine) *Repository[T] {
	return &Repository[T]{Engine: engine}
}

func (r *Repository[T]) Get(ctx context.Context, id int64) (T, error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return t, errors.New("lorm.Repository.Get() only supports tables with single-column primary keys")
	}
	return r.GetByField(ctx, primaryKeys[0], id)
}

func (r *Repository[T]) GetByField(ctx context.Context, field string, value any) (T, error) {
	return Query[T](r.Engine).
		Where(builder.Eq{field: value}).
		Get(ctx)
}

func (r *Repository[T]) Lock(ctx context.Context, id int64) (T, error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return t, errors.New("lorm.Repository.Lock() only supports tables with single-column primary keys")
	}
	return r.LockByField(ctx, primaryKeys[0], id)
}

func (r *Repository[T]) LockByField(ctx context.Context, field string, value any) (T, error) {
	q := Query[T](r.Engine).Where(builder.Eq{field: value})

	// SQLite does not support FOR UPDATE syntax
	// In SQLite, row-level locking is automatic within transactions
	switch r.Engine.DriverName() {
	case "sqlite", "sqlite3", "ql":
		// No FOR UPDATE needed for SQLite
	default:
		q = q.Suffix("FOR UPDATE")
	}

	return q.Get(ctx)
}

func (r *Repository[T]) Exist(ctx context.Context, id int64) (bool, error) {
	return r.ExistByField(ctx, "id", id)
}

func (r *Repository[T]) ExistByField(ctx context.Context, field string, value any) (bool, error) {
	return Query[T](r.Engine).
		Where(builder.Eq{field: value}).
		Exist(ctx)
}

func (r *Repository[T]) Update(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Update[T](r.Engine).SetModel(model).Exec(ctx)
}
func (r *Repository[T]) UpdateMap(ctx context.Context, id int64, data map[string]any) (rowsAffected int64, err error) {
	var table T
	return Update[T](r.Engine).
		Table(table.TableName()).
		ID(id).
		SetMap(data).
		Exec(ctx)
}

func (r *Repository[T]) Insert(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).AddModel(model).Exec(ctx)
}

func (r *Repository[T]) InsertAll(ctx context.Context, models []T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).AddModels(models...).Exec(ctx)
}

func (r *Repository[T]) InsertIgnore(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).Ignore().AddModel(model).Exec(ctx)
}

func (r *Repository[T]) InsertIgnoreAll(ctx context.Context, models []T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).Ignore().AddModels(models...).Exec(ctx)
}

func (r *Repository[T]) Delete(ctx context.Context, id int64) (rowsAffected int64, err error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return 0, errors.New("lorm.Repository.Delete() only supports tables with single-column primary keys")
	}
	return r.DeleteByField(ctx, primaryKeys[0], id)
}

func (r *Repository[T]) DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error) {
	var table T
	return Delete(r.Engine).
		From(table.TableName()).
		Where(builder.Eq{field: value}).
		Exec(ctx)
}
