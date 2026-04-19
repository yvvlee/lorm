package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
)

func TestUpdateWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = Update[*Test](e).
		Prefix("/*pre*/").
		PrefixExpr(builder.Expr("/*prex*/")).
		Set("str", "x").
		SetMap(map[string]any{"str": "y"}).
		Where("id = ?", 1).
		ID(1).
		OrderBy("id").
		Limit(10).
		Offset(0).
		Suffix("/*suf*/").
		SuffixExpr(builder.Expr("/*sufx*/"))
}

func TestUpdateExecErrorWithInvalidPrefix(t *testing.T) {
	e := initEngine(t)
	defer e.Close()
	_, err := Update[*Test](e).Prefix("INVALID").Set("str", "x").Where("id = ?", -1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecError_NoSet(t *testing.T) {
	e := initEngine(t)
	_, err := Update[*Test](e).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateSetModelRequiresPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := Update[*testNoPrimaryKeyModel](e).
		SetModel(&testNoPrimaryKeyModel{Name: "unsafe"}).
		Exec(context.TODO())
	assert.ErrorContains(t, err, "primary key")
}

func TestRepositoryUpdateRequiresPrimaryKey(t *testing.T) {
	repo := NewRepository[*testNoPrimaryKeyModel](&Engine{config: &Config{}})
	_, err := repo.Update(context.TODO(), &testNoPrimaryKeyModel{Name: "unsafe"})
	assert.ErrorContains(t, err, "primary key")
}

type testNoPrimaryKeyModel struct {
	UnimplementedTable
	Name string
}

func (m *testNoPrimaryKeyModel) TableName() string {
	return "test_no_primary_key"
}

func (m *testNoPrimaryKeyModel) New() Model {
	return new(testNoPrimaryKeyModel)
}

func (m *testNoPrimaryKeyModel) LormFieldPtr(name string) any {
	switch name {
	case "name":
		return &m.Name
	default:
		return nil
	}
}

func (m *testNoPrimaryKeyModel) LormModelDescriptor() *ModelDescriptor {
	return &ModelDescriptor{
		Name:      "testNoPrimaryKeyModel",
		TableName: m.TableName(),
		Fields: []*FieldDescriptor{
			{
				Name:     "Name",
				FullName: "Name",
				DBField:  "name",
			},
		},
	}
}
