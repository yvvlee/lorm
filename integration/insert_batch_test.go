package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/builder"
)

type batchInsertModel struct {
	lorm.UnimplementedTable
	ID         int64
	Payload    string
	expression builder.Sqlizer
}

var batchInsertDescriptor = &lorm.ModelDescriptor{
	Name: "batchInsertModel", TableName: "batch_insert_models", PrimaryKeys: []string{"id"},
	Fields: []*lorm.FieldDescriptor{
		{Name: "ID", FullName: "ID", DBField: "id", Flag: lorm.FlagPrimaryKey},
		{Name: "Payload", FullName: "Payload", DBField: "payload"},
	},
}
var batchInsertColumns = []string{"id", "payload"}

func (*batchInsertModel) TableName() string                          { return "batch_insert_models" }
func (*batchInsertModel) New() lorm.Model                            { return new(batchInsertModel) }
func (*batchInsertModel) LormModelDescriptor() *lorm.ModelDescriptor { return batchInsertDescriptor }
func (m *batchInsertModel) LormFieldPtr(name string) any {
	switch name {
	case "id":
		return &m.ID
	case "payload":
		return &m.Payload
	default:
		return nil
	}
}
func (m *batchInsertModel) LormBeforeInsert(lorm.HookTime) lorm.InsertPlan {
	var value any = m.Payload
	if m.expression != nil {
		value = m.expression
	}
	return lorm.InsertPlan{Columns: batchInsertColumns, Values: []any{m.ID, value}}
}

func newBatchInsertEngine(t *testing.T) *lorm.Engine {
	t.Helper()
	driver, dsn := mustIntegrationDriverAndDSN(t)
	engine, err := lorm.NewEngine(driver, dsn, lorm.WithMaxOpenConns(1))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close()) })
	_, err = engine.Exec(context.Background(), "DROP TABLE IF EXISTS batch_insert_models")
	require.NoError(t, err)
	_, err = engine.Exec(context.Background(), "CREATE TABLE batch_insert_models (id BIGINT PRIMARY KEY, payload VARCHAR(80) NOT NULL)")
	require.NoError(t, err)
	return engine
}

func TestInsertDefaultBatchesExceedSingleStatementParameterLimit(t *testing.T) {
	engine := newBatchInsertEngine(t)
	ctx := context.Background()
	models := make([]*batchInsertModel, 40001)
	for i := range models {
		models[i] = &batchInsertModel{ID: int64(i + 1), Payload: "batch"}
	}
	rows, err := engine.Insert[*batchInsertModel]().AddModels(models...).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, len(models), rows)
	count, err := engine.Query[*batchInsertModel]().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, len(models), count)
}

func TestInsertBatchesRollbackOnLaterFailure(t *testing.T) {
	for _, nested := range []bool{false, true} {
		for _, emptyExpression := range []bool{false, true} {
			t.Run(testBatchName(nested, emptyExpression), func(t *testing.T) {
				engine := newBatchInsertEngine(t)
				ctx := context.Background()
				last := &batchInsertModel{ID: 1, Payload: "duplicate"}
				if emptyExpression {
					last.ID, last.expression = 3, builder.Or{}
				}
				insert := func(ctx context.Context) error {
					rows, err := engine.Insert[*batchInsertModel]().BatchSize(2).AddModels(
						&batchInsertModel{ID: 1, Payload: "first"},
						&batchInsertModel{ID: 2, Payload: "second"}, last,
					).Exec(ctx)
					assert.Zero(t, rows)
					return err
				}
				var err error
				if nested {
					err = engine.TX(ctx, insert)
				} else {
					err = insert(ctx)
				}
				require.Error(t, err)
				if emptyExpression {
					assert.ErrorContains(t, err, "must not be empty")
				}
				count, err := engine.Query[*batchInsertModel]().Count(ctx)
				require.NoError(t, err)
				assert.Zero(t, count)
			})
		}
	}
}

func testBatchName(nested, empty bool) string {
	name := "own_transaction"
	if nested {
		name = "existing_transaction"
	}
	if empty {
		return name + "/empty_expression"
	}
	return name + "/duplicate_key"
}

func TestInsertBatchesIgnoreCountsOnlyInsertedRows(t *testing.T) {
	engine := newBatchInsertEngine(t)
	ctx := context.Background()
	rows, err := engine.Insert[*batchInsertModel]().BatchSize(2).Ignore().AddModels(
		&batchInsertModel{ID: 1}, &batchInsertModel{ID: 2}, &batchInsertModel{ID: 1}, &batchInsertModel{ID: 3},
	).Exec(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 3, rows)
	count, err := engine.Query[*batchInsertModel]().Count(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
}
