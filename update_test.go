package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestUpdateTypedNilReturnsErrorFromExec(t *testing.T) {
	stmt := (&Engine{config: &Config{}}).Update[*Test]().SetModel(nil)

	_, err := stmt.Exec(context.Background())
	assert.ErrorContains(t, err, "model is nil")
}

func TestUpdateAssignmentModesCannotBeMixed(t *testing.T) {
	engine := &Engine{config: &Config{}}
	model := &reservedWordModel{ID: 1, Group: "model"}

	tests := []struct {
		name string
		mode updateMode
		stmt *UpdateStmt[*reservedWordModel]
	}{
		{
			name: "model then set",
			mode: updateModeModel,
			stmt: engine.Update[*reservedWordModel]().
				SetModel(model).
				Set("group", "manual"),
		},
		{
			name: "model then set map",
			mode: updateModeModel,
			stmt: engine.Update[*reservedWordModel]().
				SetModel(model).
				SetMap(map[string]any{"group": "manual"}),
		},
		{
			name: "set then model",
			mode: updateModeManual,
			stmt: engine.Update[*reservedWordModel]().
				Set("group", "manual").
				SetModel(model),
		},
		{
			name: "set map then model",
			mode: updateModeManual,
			stmt: engine.Update[*reservedWordModel]().
				SetMap(map[string]any{"group": "manual"}).
				SetModel(model),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.mode, tt.stmt.mode)
			_, err := tt.stmt.Exec(context.Background())
			assert.ErrorContains(t, err, "cannot be mixed")
		})
	}
}

func TestUpdateSetModelCanOnlyBeCalledOnce(t *testing.T) {
	model := &reservedWordModel{ID: 1, Group: "model"}
	stmt := (&Engine{config: &Config{}}).
		Update[*reservedWordModel]().
		SetModel(model).
		SetModel(model)

	_, err := stmt.Exec(context.Background())
	assert.ErrorContains(t, err, "only be called once")
}

func TestUpdateCloneCopiesModeAndFirstError(t *testing.T) {
	stmt := (&Engine{config: &Config{}}).
		Update[*reservedWordModel]().
		Set("group", "manual").
		SetModel(&reservedWordModel{ID: 1})
	clone := stmt.Clone()

	assert.Equal(t, updateModeManual, clone.mode)
	assert.EqualError(t, clone.err, stmt.err.Error())
	clone.SetModel(&reservedWordModel{ID: 2})
	assert.EqualError(t, clone.err, stmt.err.Error())
}

func TestUpdateRequiresWhereUnlessGlobalWriteIsExplicit(t *testing.T) {
	recorder := newCaptureSQLRecorder()
	engine := newCaptureSQLEngine(t, recorder, false, testLogger{})

	_, err := engine.Update[*reservedWordModel]().Set("group", "all").Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause or AllowGlobalWrite")
	assert.Empty(t, recorder.Calls())

	_, err = engine.Update[*reservedWordModel]().
		Set("group", "all").
		Where(builder.Eq{}).
		Exec(context.Background())
	assert.ErrorContains(t, err, "requires a WHERE clause or AllowGlobalWrite")
	assert.Empty(t, recorder.Calls())

	rows, err := engine.Update[*reservedWordModel]().
		Set("group", "all").
		AllowGlobalWrite().
		Exec(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 1, rows)
	assert.Equal(t, "UPDATE `order` SET `group` = ?", recorder.Last().query)
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
