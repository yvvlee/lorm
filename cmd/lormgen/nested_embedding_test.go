package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yvvlee/lorm/names"
)

func TestGenerateNestedEmbeddedPointersAcrossPackages(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	dir := t.TempDir()
	t.Chdir(dir)
	previousWd := wd
	wd = dir
	t.Cleanup(func() { wd = previousWd })
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/nested\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
	copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
	writeFile(t, filepath.Join(dir, "inner", "inner.go"), `package inner
import (
 "time"
 "github.com/yvvlee/lorm"
)
type hidden struct { Hidden string }
type Details struct {
 Label string
 private func()
 *hidden
}
type Inner struct {
 lorm.UnimplementedModel
 Details
 ID int64 `+"`lorm:\"id,primary_key,auto_increment\"`"+`
 Name string
 Data []string `+"`lorm:\"data,json\"`"+`
 Updated time.Time `+"`lorm:\"updated,updated\"`"+`
 Version int64 `+"`lorm:\"version,version\"`"+`
 privateVersion int8 `+"`lorm:\"version\"`"+`
}
`)
	writeFile(t, filepath.Join(dir, "outer", "outer.go"), `package outer
import "example.com/nested/inner"
type Outer struct {
 *inner.Inner `+"`lorm:\"inner_\"`"+`
 Code string
 privateCode string
}
`)
	model := filepath.Join(dir, "model.go")
	writeFile(t, model, `package nested
import (
 b "example.com/nested/outer"
 "github.com/yvvlee/lorm"
)
type Model struct {
 lorm.UnimplementedTable
 *b.Outer `+"`lorm:\"outer_\"`"+`
 private func()
}
`)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	output, err := tidy.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, g.Generate([]string{model}))
	require.FileExists(t, filepath.Join(dir, "model_lorm_gen.go"))
	writeFile(t, filepath.Join(dir, "model_test.go"), `package nested
import (
 "database/sql"
 "reflect"
 "testing"
 "time"
 b "example.com/nested/outer"
 "github.com/yvvlee/lorm"
)
type scanRow struct{}
func (scanRow) Scan(dst ...any) error {
 values := []any{"label", int64(7), "name", []byte("[\"json\"]"), time.Unix(100, 0), int64(2), "code"}
 for i, value := range values {
  if scanner, ok := dst[i].(sql.Scanner); ok {
   if err := scanner.Scan(value); err != nil { return err }
  } else { reflect.ValueOf(dst[i]).Elem().Set(reflect.ValueOf(value)) }
 }
 return nil
}
func TestNested(t *testing.T) {
 var m Model
 want := []string{"outer_inner_label", "outer_inner_id", "outer_inner_name", "outer_inner_data", "outer_inner_updated", "outer_inner_version", "outer_code"}
 if got := m.LormCols().All(); !reflect.DeepEqual(got, want) { t.Fatal(got) }
 if m.LormFieldValue("outer_inner_name") != "" || m.Outer == nil || m.Inner == nil { t.Fatal("embedded values must be initialized") }
 if m.LormFieldValue("outer_inner_version") != int64(0) { t.Fatal("embedded version must use its zero value") }
 m = Model{}
 if _, err := m.LormBeforeUpdate(time.Now()); err != nil { t.Fatal(err) }
 if m.Outer == nil || m.Inner == nil || m.Version != 0 || !m.Updated.IsZero() { t.Fatal("update preparation must only initialize embedded structs") }
 m.Outer = &b.Outer{}
 if m.LormFieldValue("outer_inner_name") != "" || m.Inner == nil { t.Fatal("inner pointer must be initialized") }
 *m.LormFieldPtr("outer_inner_name").(*string) = "allocated"
 if m.Name != "allocated" { t.Fatal(m) }
 m = Model{}
 if err := m.LormScan(scanRow{}); err != nil { t.Fatal(err) }
 if m.ID != 7 || m.Name != "name" || m.Label != "label" || m.Data[0] != "json" || m.Code != "code" { t.Fatal(m) }
 now := time.Unix(200, 0)
 plan, err := m.LormBeforeUpdate(now)
 if err != nil { t.Fatal(err) }
 if len(plan.Where) != 2 || !reflect.DeepEqual(plan.Increment, []string{"outer_inner_version"}) { t.Fatal(plan) }
 m.LormAfterUpdate(now, 1)
 if m.Version != 3 || !m.Updated.Equal(now) { t.Fatal(m) }
 m = Model{}
 insert := m.LormBeforeInsert(now)
 if !insert.AutoIncrementZero || insert.AutoIncrementColumn != "outer_inner_id" || len(insert.Values) != 6 { t.Fatal(insert) }
 if m.Inner == nil || !m.Updated.Equal(now) { t.Fatal(m) }
 m = Model{}
 if err := m.LormAfterInsert(lorm.InsertResult{HasGeneratedID:true, GeneratedID:11}); err != nil { t.Fatal(err) }
 if m.ID != 11 { t.Fatal(m) }
}
`)
	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func TestGenerateRejectsRecursiveEmbedding(t *testing.T) {
	_, err := extractSource(t, `package validation
import "github.com/yvvlee/lorm"
type Loop struct { *Loop }
type Model struct {
 lorm.UnimplementedTable
 Loop
 Name string
}
`)
	require.ErrorContains(t, err, "recursive embedded field Model.Loop.Loop")
}
