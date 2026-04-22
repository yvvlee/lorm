package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

func TestTransactionFlow(t *testing.T) {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		t.Skipf("sqlite example unavailable: %v", err)
		return
	}
	defer cleanup()

	alice := &Account{Owner: "Alice", Balance: 100}
	_, err = lorm.Insert[*Account](engine).AddModel(alice).Exec(ctx)
	assert.NoError(t, err)

	bob := &Account{Owner: "Bob", Balance: 20}
	_, err = lorm.Insert[*Account](engine).AddModel(bob).Exec(ctx)
	assert.NoError(t, err)

	err = transfer(ctx, engine, alice.ID, bob.ID, 30)
	assert.NoError(t, err)

	repo := lorm.NewRepository[*Account](engine)
	aliceAfter, err := repo.Get(ctx, alice.ID)
	assert.NoError(t, err)
	assert.EqualValues(t, 70, aliceAfter.Balance)

	bobAfter, err := repo.Get(ctx, bob.ID)
	assert.NoError(t, err)
	assert.EqualValues(t, 50, bobAfter.Balance)

	err = transfer(ctx, engine, bob.ID, alice.ID, 1_000)
	assert.ErrorContains(t, err, "insufficient balance")

	bobRollback, err := repo.Get(ctx, bob.ID)
	assert.NoError(t, err)
	assert.EqualValues(t, 50, bobRollback.Balance)
}
