package lorm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFileDescriptorHelpers(t *testing.T) {
	d := &FileDescriptor{Path: "a/b/c.go"}
	prefix := d.RawVarPrefix()
	assert.Contains(t, prefix, "_lorm_file_a_b_c")

	s := d.JsonMarshal()
	assert.NotEmpty(t, s)
}
