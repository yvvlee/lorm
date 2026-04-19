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

func TestQueryIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, err := Query[*testNoPrimaryKeyModel](e).ID(1).Get(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = Query[*testCompositePrimaryKeyModel](e).ID(1).Exist(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}

func TestUpdateIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, err := Update[*testNoPrimaryKeyModel](e).
		ID(1).
		Set("name", "unsafe").
		Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = Update[*testCompositePrimaryKeyModel](e).
		ID(1).
		Set("name", "unsafe").
		Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}

func TestDeleteModelIDRequiresSingleColumnPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}

	_, err := DeleteModel[*testNoPrimaryKeyModel](e).ID(1).Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")

	_, err = DeleteModel[*testCompositePrimaryKeyModel](e).ID(1).Exec(context.TODO())
	assert.ErrorContains(t, err, "single-column primary keys")
}
