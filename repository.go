package lorm

import (
	"context"

	"github.com/yvvlee/lorm/builder"
)

type Repository[T Table] struct {
	Engine *Engine
}

func NewRepository[T Table](engine *Engine) *Repository[T] {
	return &Repository[T]{Engine: engine}
}

func (r *Repository[T]) Get(ctx context.Context, id int64) (T, error) {
	return r.GetByField(ctx, "id", id)
}

func (r *Repository[T]) GetByField(ctx context.Context, field string, value any) (T, error) {
	var t T
	return Query[T](r.Engine).
		From(t.TableName()).
		Where(builder.Eq{field: value}).
		Get(ctx)
}

func (r *Repository[T]) Lock(ctx context.Context, id int64) (T, error) {
	return r.LockByField(ctx, "id", id)
}

func (r *Repository[T]) LockByField(ctx context.Context, field string, value any) (T, error) {
	var t T
	return Query[T](r.Engine).
		From(t.TableName()).
		Where(builder.Eq{field: value}).
		Suffix("FOR UPDATE").
		Get(ctx)
}

func (r *Repository[T]) Exist(ctx context.Context, id int64) (bool, error) {
	return r.ExistByField(ctx, "id", id)
}

func (r *Repository[T]) ExistByField(ctx context.Context, field string, value any) (bool, error) {
	var t T
	return Query[T](r.Engine).
		From(t.TableName()).
		Where(builder.Eq{field: value}).
		Exist(ctx)
}

func (r *Repository[T]) Update(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Update(r.Engine).SetModel(model).Exec(ctx)
}
func (r *Repository[T]) UpdateMap(ctx context.Context, id int64, data map[string]any) (rowsAffected int64, err error) {
	var table T
	return Update(r.Engine).
		Table(table.TableName()).
		ID(id).
		SetMap(data).
		Exec(ctx)
}

func (r *Repository[T]) Insert(ctx context.Context, model T) (rowsAffected int64, err error) {
	return Insert(ctx, r.Engine, model)
}

func (r *Repository[T]) InsertAll(ctx context.Context, models []T) (rowsAffected int64, err error) {
	return InsertAll(ctx, r.Engine, models)
}

func (r *Repository[T]) Delete(ctx context.Context, id int64) (rowsAffected int64, err error) {
	return r.DeleteByField(ctx, "id", id)
}

func (r *Repository[T]) DeleteByField(ctx context.Context, field string, value any) (rowsAffected int64, err error) {
	var table T
	return Delete(r.Engine).
		From(table.TableName()).
		Where(builder.Eq{field: value}).
		Exec(ctx)
}

//
//type PageQuerierHook func(*xorm.Session) *xorm.Session
//
//var DescIDHook = func(session *xorm.Session) *xorm.Session {
//	return session.OrderBy("id desc")
//}
//
//type PageQuerier struct {
//	table     any
//	ListHook  PageQuerierHook
//	CountHook PageQuerierHook
//	Page      *common.Pagination
//	Filter    Filter
//}
//
//type Filter interface {
//	BuildCondition(db *xorm.Session) (builder.Cond, error)
//}
//
//func (b *PageQuerier) Query(db *xorm.Session, list any) error {
//	cond, err := b.Filter.BuildCondition(db)
//	if err != nil {
//		return err
//	}
//	var total int64
//	if b.Page != nil {
//		b.Page.Page = max(1, b.Page.Page)
//		if b.Page.limit < 1 {
//			b.Page.limit = 10
//		}
//		if b.CountHook != nil {
//			db = b.CountHook(db)
//		}
//		total, err = db.table(b.table).And(cond).Count()
//		if err != nil {
//			return WrapDatabaseError(err)
//		}
//		if total == 0 {
//			return nil
//		}
//		b.Page.Total = uint32(total)
//		db = db.limit(int(b.Page.limit), int((b.Page.Page-1)*b.Page.limit))
//	}
//	if b.ListHook != nil {
//		db = b.ListHook(db)
//	}
//	if err = db.table(b.table).And(cond).Find(list); err != nil {
//		return WrapDatabaseError(err)
//	}
//	return nil
//}
//
//type IDOnly struct {
//	ID int64
//}
