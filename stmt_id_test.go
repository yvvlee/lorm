package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCompositePrimaryKeyModel struct {
	UnimplementedTable
	AccountID int64
	TenantID  int64
	Name      string
}

func (m *testCompositePrimaryKeyModel) TableName() string {
	return "test_composite_primary_key"
}

func (m *testCompositePrimaryKeyModel) New() Model {
	return new(testCompositePrimaryKeyModel)
}

func (m *testCompositePrimaryKeyModel) LormFieldPtr(name string) any {
	switch name {
	case "account_id":
		return &m.AccountID
	case "tenant_id":
		return &m.TenantID
	case "name":
		return &m.Name
	default:
		return nil
	}
}

func (m *testCompositePrimaryKeyModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "testCompositePrimaryKeyModel",
		TableName: m.TableName(),
		Fields: []*FieldDescriptor{
			{
				Name:     "AccountID",
				FullName: "AccountID",
				DBField:  "account_id",
				Flag:     FlagPrimaryKey,
			},
			{
				Name:     "TenantID",
				FullName: "TenantID",
				DBField:  "tenant_id",
				Flag:     FlagPrimaryKey,
			},
			{
				Name:     "Name",
				FullName: "Name",
				DBField:  "name",
			},
		},
	}
}

func TestSelectIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, _, err := e.Select[*testNoPrimaryKeyModel]().ID(1).Get(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = e.Select[*testCompositePrimaryKeyModel]().ID(1).Exist(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}

func TestSelectIDRequiresModelResult(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, _, err := e.Select[int64]().ID(1).Get(context.TODO())
	assert.ErrorContains(t, err, "requires a Model result type")
}

func TestUpdateIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, err := e.Update[*testNoPrimaryKeyModel]().
		ID(1).
		Set("name", "unsafe").
		Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = e.Update[*testCompositePrimaryKeyModel]().
		ID(1).
		Set("name", "unsafe").
		Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}

func TestDeleteIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, err := e.Delete[*testNoPrimaryKeyModel]().ID(1).Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = e.Delete[*testCompositePrimaryKeyModel]().ID(1).Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}
