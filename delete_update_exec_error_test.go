package lorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteExecError_NoFrom(t *testing.T) {
	e := initEngine(t)
	_, err := Delete(e).Where("id = ?", 1).Exec(context.TODO())
	assert.Error(t, err)
}

func TestUpdateExecError_NoSet(t *testing.T) {
	e := initEngine(t)
	_, err := Update[*Test](e).Exec(context.TODO())
	assert.Error(t, err)
}
