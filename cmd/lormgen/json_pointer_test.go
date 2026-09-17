package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yvvlee/lorm/names"
)

func TestGeneratedJSONPointerFieldsUseJSON(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	repo := findRepoRoot(t, cwd)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/jsonpointer\n\ngo 1.27.1\n\nrequire github.com/yvvlee/lorm v0.0.0\nreplace github.com/yvvlee/lorm => "+filepath.ToSlash(repo)+"\n")
	copyFile(t, filepath.Join(repo, "go.sum"), filepath.Join(dir, "go.sum"))
	model := filepath.Join(dir, "model.go")
	writeFile(t, model, `package jsonpointer
import (
 "database/sql/driver"
 "errors"
 "github.com/yvvlee/lorm"
)
var conversionErr = errors.New("custom conversion failed")
type Codec struct { Text string }
func (*Codec) Scan(any) error { return conversionErr }
func (*Codec) Value() (driver.Value, error) { return nil, conversionErr }
type Alias = *Codec
type Payload struct { Name string }
type NamedPointer *Payload
type User struct {
 lorm.UnimplementedTable "lorm:\"custom_users\""
 ID int64 "lorm:\"user_id,primary_key\""
 Data *Codec "lorm:\"data,json\""
 Aliased Alias "lorm:\"aliased,json\""
 Plain NamedPointer "lorm:\"plain,json\""
 Native *Codec "lorm:\"native\""
 Created int64 "lorm:\"created\""
}
type Keyed struct {
 lorm.UnimplementedTable
 Key *Codec "lorm:\"key,json,primary_key\""
 Name string
}
`)
	g := NewGenerator(new(names.SnakeMapper), new(names.SnakeMapper), "lorm", "")
	require.NoError(t, g.Generate([]string{model}))
	writeFile(t, filepath.Join(dir, "model_test.go"), `package jsonpointer
import (
 "database/sql"
 "database/sql/driver"
 "errors"
 "testing"
 "time"
)
type row struct{ source any }
func (r row) Scan(dest ...any) error { return dest[1].(sql.Scanner).Scan(r.source) }
func assertJSONValue(t *testing.T, value any, want string) {
 t.Helper()
 encoded, err := value.(driver.Valuer).Value()
 if err != nil { t.Fatal(err) }
 if string(encoded.([]byte)) != want { t.Fatalf("unexpected JSON: %s", encoded) }
}
func TestGeneratedConversions(t *testing.T) {
 model := new(User)
 if model.TableName() != "custom_users" || model.LormModelDescriptor().PrimaryKeys[0] != "user_id" {
  t.Fatal("interpreted tags were lost")
 }
 if err := model.LormFieldPtr("data").(sql.Scanner).Scan("not JSON"); err == nil || errors.Is(err, conversionErr) {
  t.Fatalf("named scan must reject invalid JSON without calling Codec.Scan: %v", err)
 }
 if model.Data != nil { t.Fatal("failed named scan initialized field") }
 if err := model.LormScan(row{source: "not JSON"}); err == nil || errors.Is(err, conversionErr) {
  t.Fatalf("ordered scan must reject invalid JSON without calling Codec.Scan: %v", err)
 }
 if err := model.LormScan(row{source: "{\"Text\":\"json\"}"}); err != nil { t.Fatal(err) }
 if model.Data == nil || model.Data.Text != "json" { t.Fatal("JSON pointer was not initialized") }
 assertJSONValue(t, model.LormFieldValue("data"), "{\"Text\":\"json\"}")
 insert := model.LormBeforeInsert(time.Unix(100, 0))
 assertJSONValue(t, insert.Values[1], "{\"Text\":\"json\"}")
 update, err := model.LormBeforeUpdate(time.Unix(200, 0))
 if err != nil { t.Fatal(err) }
 if update.Set[0].Column != "data" { t.Fatal(update) }
 assertJSONValue(t, update.Set[0].Value, "{\"Text\":\"json\"}")
 keyed := &Keyed{Key: &Codec{Text: "key"}}
 keyUpdate, err := keyed.LormBeforeUpdate(time.Time{})
 if err != nil { t.Fatal(err) }
 assertJSONValue(t, keyUpdate.Where[0].Value, "{\"Text\":\"key\"}")
 if err := model.LormFieldPtr("aliased").(sql.Scanner).Scan("{\"Text\":\"alias\"}"); err != nil { t.Fatal(err) }
 if model.Aliased == nil || model.Aliased.Text != "alias" { t.Fatal("pointer alias was not decoded") }
 if err := model.LormFieldPtr("plain").(sql.Scanner).Scan("{\"Name\":\"plain\"}"); err != nil { t.Fatal(err) }
 if model.Plain == nil || model.Plain.Name != "plain" { t.Fatal("named pointer JSON decoding failed") }
 if _, ok := model.LormFieldPtr("native").(**Codec); !ok { t.Fatal("unmarked pointer field was wrapped") }
 model.Native = &Codec{Text: "native"}
 if _, err := model.LormFieldValue("native").(driver.Valuer).Value(); !errors.Is(err, conversionErr) {
  t.Fatalf("unmarked field lost its database interface: %v", err)
 }
}
`)
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
