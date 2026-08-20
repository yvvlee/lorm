package lorm

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	benchmarkPrimaryKeyFields []string
)

func TestFileDescriptorHelpers(t *testing.T) {
	d := &FileDescriptor{Path: "a/b/c.go"}
	prefix := d.RawVarPrefix()
	assert.Contains(t, prefix, "_lorm_file_a_b_c")
	assert.True(t, token.IsIdentifier(prefix))
	assert.NotEqual(t,
		(&FileDescriptor{Path: "feature-v2/model.go"}).RawVarPrefix(),
		(&FileDescriptor{Path: "feature_v2/model.go"}).RawVarPrefix(),
	)

	s := d.JsonMarshal()
	assert.NotEmpty(t, s)
}

func TestGeneratedModelDescriptorIsCached(t *testing.T) {
	var model *Test
	descriptor := model.LormModelDescriptor()

	assert.Same(t, descriptor, model.LormModelDescriptor())
	assert.Equal(t, []string{"id"}, descriptor.PrimaryKeys)
}

func BenchmarkPrimaryKeyLookup(b *testing.B) {
	var model *Test
	descriptor := model.LormModelDescriptor()

	b.Run("CachedDescriptor", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkPrimaryKeyFields = model.LormModelDescriptor().PrimaryKeys
		}
	})

	b.Run("DescriptorScan", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkPrimaryKeyFields = descriptor.FlagFields(FlagPrimaryKey)
		}
	})
}
