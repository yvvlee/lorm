package lorm

import (
	"context"
	"errors"
	"fmt"

	"github.com/yvvlee/lorm/builder"
)

// Repository provides common CRUD helpers for table T.
type Repository[T Table] struct {
	Engine *Engine
}

// NewRepository creates a Repository backed by engine.
func NewRepository[T Table](engine *Engine) *Repository[T] {
	return &Repository[T]{Engine: engine}
}

// Get loads a row by its single-column primary key.
func (r *Repository[T]) Get(ctx context.Context, id any) (T, error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return t, errors.New("lorm.Repository.Get() only supports tables with single-column primary keys")
	}
	return r.GetByField(ctx, primaryKeys[0], id)
}

// GetByField loads the first row matching field = value.
func (r *Repository[T]) GetByField(ctx context.Context, field string, value any) (T, error) {
	return Query[T](r.Engine).
		Where(builder.Eq{field: value}).
		Get(ctx)
}

// Lock loads a row by primary key and appends FOR UPDATE when supported.
func (r *Repository[T]) Lock(ctx context.Context, id any) (T, error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return t, errors.New("lorm.Repository.Lock() only supports tables with single-column primary keys")
	}
	return r.LockByField(ctx, primaryKeys[0], id)
}

// LockByField loads the first row matching field = value and locks it for update.
func (r *Repository[T]) LockByField(ctx context.Context, field string, value any) (T, error) {
	var t T
	if !r.Engine.SupportsForUpdate() {
		return t, fmt.Errorf("lorm.Repository.LockByField() does not support FOR UPDATE for driver %q", r.Engine.DriverName())
	}
	return Query[T](r.Engine).
		Where(builder.Eq{field: value}).
		Suffix("FOR UPDATE").
		Get(ctx)
}

// Exist reports whether a row with the given primary key exists.
func (r *Repository[T]) Exist(ctx context.Context, id any) (bool, error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return false, errors.New("lorm.Repository.Exist() only supports tables with single-column primary keys")
	}
	return r.ExistByField(ctx, primaryKeys[0], id)
}

// ExistByField reports whether any row matches field = value.
func (r *Repository[T]) ExistByField(ctx context.Context, field string, value any) (bool, error) {
	return Query[T](r.Engine).
		Where(builder.Eq{field: value}).
		Exist(ctx)
}

// Update updates model using its primary key fields.
func (r *Repository[T]) Update(ctx context.Context, model T) (rowsAffected int64, err error) {
	var t T
	if len(t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)) == 0 {
		return 0, errors.New("lorm.Repository.Update() requires tables with at least one primary key")
	}
	return Update[T](r.Engine).SetModel(model).Exec(ctx)
}

// UpdateMap updates the row identified by id with the provided column values.
func (r *Repository[T]) UpdateMap(ctx context.Context, id any, data map[string]any) (rowsAffected int64, err error) {
	return Update[T](r.Engine).
		ID(id).
		SetMap(data).
		Exec(ctx)
}

// Insert inserts a model.
func (r *Repository[T]) Insert(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).AddModel(model).Exec(ctx)
}

// InsertAll inserts models in one batch.
func (r *Repository[T]) InsertAll(ctx context.Context, models []T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).AddModels(models...).Exec(ctx)
}

// InsertIgnore inserts a model while ignoring duplicate conflicts when supported.
func (r *Repository[T]) InsertIgnore(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).Ignore().AddModel(model).Exec(ctx)
}

// InsertIgnoreAll inserts models while ignoring duplicate conflicts when supported.
func (r *Repository[T]) InsertIgnoreAll(ctx context.Context, models []T) (rowsAffected int64, err error) {
	return Insert[T](r.Engine).Ignore().AddModels(models...).Exec(ctx)
}

// Delete deletes a row by its single-column primary key.
func (r *Repository[T]) Delete(ctx context.Context, id any) (rowsAffected int64, err error) {
	var t T
	primaryKeys := t.New().LormModelDescriptor().FlagFields(FlagPrimaryKey)
	if len(primaryKeys) != 1 {
		return 0, errors.New("lorm.Repository.Delete() only supports tables with single-column primary keys")
	}
	return r.DeleteByField(ctx, primaryKeys[0], id)
}

// DeleteByField deletes rows matching field = value.
func (r *Repository[T]) DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error) {
	return DeleteModel[T](r.Engine).
		Where(builder.Eq{field: value}).
		Exec(ctx)
}
