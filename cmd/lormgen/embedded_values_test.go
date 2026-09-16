package main

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/names"
)

func TestGeneratedEmbeddedValuesUseFlattenedFieldSemantics(t *testing.T) {
	info, err := extractSource(t, `package validation
import (
 "time"
 "github.com/yvvlee/lorm"
)
type Metadata struct { Value string }
type Leaf struct {
 Name string
 Age int64
 Enabled bool
 Optional *string
 Meta Metadata `+"`lorm:\"meta,json\"`"+`
}
type Middle struct { *Leaf }
type Manual struct {
 lorm.UnimplementedTable
 ID int64 `+"`lorm:\"id,primary_key\"`"+`
 *Middle
 Direct *string
}
type Automatic struct {
 lorm.UnimplementedTable
 ID int64 `+"`lorm:\"id,primary_key,auto_increment\"`"+`
 *Middle
 Direct *string
}
type Key struct { ID int64 `+"`lorm:\"id,primary_key\"`"+` }
type EmbeddedKey struct { lorm.UnimplementedTable; *Key; Name string }
type VersionFields struct {
 Revision int64 `+"`lorm:\"revision,version\"`"+`
 Updated time.Time `+"`lorm:\"updated,updated\"`"+`
}
type Versioned struct { lorm.UnimplementedTable; ID int64 `+"`lorm:\"id,primary_key\"`"+`; *VersionFields; Name string }
type PointerVersionFields struct { Revision *int64 `+"`lorm:\"revision,version\"`"+` }
type PointerVersion struct { lorm.UnimplementedTable; ID int64 `+"`lorm:\"id,primary_key\"`"+`; *PointerVersionFields; Name string }
`)
	require.NoError(t, err)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	_, err = g.generateFile(info)
	require.NoError(t, err)
	writeFile(t, filepath.Join(filepath.Dir(info.Path), "model_test.go"), `package validation
import (
 "context"
 "database/sql"
 "database/sql/driver"
 "errors"
 "reflect"
 "testing"
 "time"
 "github.com/yvvlee/lorm"
)
type recordingDriver struct { args []any }
type recordingConn struct { recorder *recordingDriver }
func (d *recordingDriver) Open(string) (driver.Conn, error) { return &recordingConn{d}, nil }
func (*recordingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unexpected Prepare") }
func (*recordingConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected Begin") }
func (*recordingConn) Close() error { return nil }
func (c *recordingConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
 c.recorder.args = make([]any, len(args))
 for i, arg := range args { c.recorder.args[i] = arg.Value }
 return driver.RowsAffected(1), nil
}
func TestWrites(t *testing.T) {
 recorder := new(recordingDriver)
 sql.Register("embedded_values", recorder)
 engine, err := lorm.NewEngine("embedded_values", "", lorm.WithLogger(nil))
 if err != nil { t.Fatal(err) }
 defer engine.Close()
 checkWrites(t, engine, recorder, func() *Manual { return &Manual{ID: 1} }, func(m *Manual) bool {
  return m.Middle != nil && m.Leaf != nil && m.Optional == nil && m.Direct == nil
 })
 checkWrites(t, engine, recorder, func() *Automatic { return &Automatic{ID: 1} }, func(m *Automatic) bool {
  return m.Middle != nil && m.Leaf != nil && m.Optional == nil && m.Direct == nil
 })
}
func checkWrites[P lorm.TablePointer[M], M any](t *testing.T, engine *lorm.Engine, recorder *recordingDriver, create func() P, initialized func(P) bool) {
 t.Helper()
 for _, update := range []bool{false, true} {
  model := create()
  var count int64
  var err error
  want := []any{int64(1), "", int64(0), false, nil, []byte("{\"Value\":\"\"}"), nil}
  if update {
   count, err = engine.Update[P]().SetModel(model).Exec(context.Background())
   want = append(want[1:], int64(1))
  } else {
   count, err = engine.Insert[P]().AddModel(model).Exec(context.Background())
  }
  if err != nil { t.Fatal(err) }
  if count != 1 || !initialized(model) || !reflect.DeepEqual(recorder.args, want) {
   t.Fatalf("%T update=%v: count=%d initialized=%v args=%#v want=%#v", model, update, count, initialized(model), recorder.args, want)
  }
 }
}
func TestFieldValues(t *testing.T) {
 var m Manual
 if m.LormFieldValue("direct") != nil || m.Middle != nil { t.Fatal("ordinary pointers must stay nil") }
 if m.LormFieldValue("name") != "" || m.Middle == nil || m.Leaf == nil { t.Fatal("initialize embedded parents") }
 if m.LormFieldValue("age") != int64(0) || m.LormFieldValue("enabled") != false { t.Fatal("use leaf zero values") }
 if m.LormFieldValue("optional") != nil || m.Optional != nil { t.Fatal("ordinary pointers inside embeds stay nil") }
 m.Name = "alice"
 m.Age = 42
 m.Enabled = true
 value := "present"
 m.Optional = &value
 middle, leaf := m.Middle, m.Leaf
 if m.LormFieldValue("name") != "alice" || m.LormFieldValue("optional") != m.Optional || m.Middle != middle || m.Leaf != leaf { t.Fatal("preserve existing values and parents") }
 plan, err := m.LormBeforeUpdate(time.Now())
 if err != nil || plan.Set[0].Value != "alice" || plan.Set[1].Value != int64(42) || plan.Set[2].Value != true || plan.Set[3].Value != m.Optional { t.Fatal(plan, err) }
}
func TestEmbeddedPrimaryKeyAndVersion(t *testing.T) {
 var key EmbeddedKey
 plan, err := key.LormBeforeUpdate(time.Now())
 if err != nil || key.Key == nil || plan.PrimaryKeyCount != 1 || plan.Where[0].Value != int64(0) { t.Fatal(plan, err) }
 var m Versioned
 now := time.Unix(123, 0)
 plan, err = m.LormBeforeUpdate(now)
 if err != nil || m.VersionFields == nil || plan.Where[1].Value != int64(0) || m.Revision != 0 || !m.Updated.IsZero() { t.Fatal(plan, err) }
 m.LormAfterUpdate(now, 0)
 if m.Revision != 0 || !m.Updated.IsZero() { t.Fatal("no managed value backfill on a miss") }
 m.LormAfterUpdate(now, 1)
 if m.Revision != 1 || m.Updated != now { t.Fatal(m) }
 var pointer PointerVersion
 if _, err := pointer.LormBeforeUpdate(now); err == nil { t.Fatal("nil pointer version must still fail") }
 if pointer.PointerVersionFields == nil || pointer.Revision != nil { t.Fatal("only allocate embedded parents") }
 revision := int64(0)
 pointer.Revision = &revision
 if _, err := pointer.LormBeforeUpdate(now); err != nil { t.Fatal(err) }
 pointer.LormAfterUpdate(now, 1)
 if revision != 1 { t.Fatal(revision) }
}
`)
	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = filepath.Dir(info.Path)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
