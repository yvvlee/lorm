package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm/builder"
)

func TestUpdateWrappers(t *testing.T) {
	e := &Engine{config: &Config{}}
	_ = e.Update[*Test]().
		Table("test_archive").
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

func TestUpdateExecError_NoSet(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := e.Update[*Test]().Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateSetModelRequiresPrimaryKey(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := e.Update[*testNoPrimaryKeyModel]().
		SetModel(&testNoPrimaryKeyModel{Name: "unsafe"}).
		Exec(context.TODO())
	assert.ErrorContains(t, err, "primary key")
}

func TestRepositoryUpdateRequiresPrimaryKey(t *testing.T) {
	repo := (&Engine{config: &Config{}}).Repository[*testNoPrimaryKeyModel]()
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
