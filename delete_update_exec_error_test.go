package lorm

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleteExecError_NoFrom(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := Delete(e).Where("id = ?", 1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecError_NoTable(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := Update(e).Set("str", "x").Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecError_NoSet(t *testing.T) {
	e := &Engine{config: &Config{}}
	_, err := Update(e).Table("test").Exec(context.TODO())
	assert.Error(t, err)
}
