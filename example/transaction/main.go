package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"

	"github.com/yvvlee/lorm"
	"github.com/yvvlee/lorm/example/internal/exampleutil"
)

//go:embed schema.sql
var schemaSQL string

func main() {
	ctx := context.Background()
	engine, cleanup, err := exampleutil.NewSQLiteEngine(schemaSQL)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	alice := &Account{Owner: "Alice", Balance: 100}
	if _, err := engine.Insert[*Account]().AddModel(alice).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	bob := &Account{Owner: "Bob", Balance: 20}
	if _, err := engine.Insert[*Account]().AddModel(bob).Exec(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("before transfer:")
	printAccounts(ctx, engine)

	if err := transfer(ctx, engine, alice.ID, bob.ID, 30); err != nil {
		log.Fatal(err)
	}

	fmt.Println("after successful transfer:")
	printAccounts(ctx, engine)

	if err := transfer(ctx, engine, bob.ID, alice.ID, 1_000); err != nil {
		fmt.Printf("rollback example: %v\n", err)
	}

	fmt.Println("after failed transfer:")
	printAccounts(ctx, engine)
}

func transfer(ctx context.Context, engine *lorm.Engine, fromID, toID, amount int64) error {
	repo := engine.Repository[*Account]()
	var a Account

	return engine.TX(ctx, func(txCtx context.Context) error {
		from, err := repo.Get(txCtx, fromID)
		if err != nil {
			return err
		}
		if from == nil {
			return fmt.Errorf("source account %d not found", fromID)
		}

		to, err := repo.Get(txCtx, toID)
		if err != nil {
			return err
		}
		if to == nil {
			return fmt.Errorf("destination account %d not found", toID)
		}

		if from.Balance < amount {
			return errors.New("insufficient balance")
		}

		if _, err := engine.Update[*Account]().
			ID(from.ID).
			SetMap(map[string]any{
				a.Fields().Balance(): from.Balance - amount,
			}).
			Exec(txCtx); err != nil {
			return err
		}

		if _, err := engine.Update[*Account]().
			ID(to.ID).
			SetMap(map[string]any{
				a.Fields().Balance(): to.Balance + amount,
			}).
			Exec(txCtx); err != nil {
			return err
		}

		return nil
	})
}

func printAccounts(ctx context.Context, engine *lorm.Engine) {
	var a Account
	accounts, err := engine.Query[*Account]().
		OrderBy(a.Fields().ID() + " ASC").
		Find(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, item := range accounts {
		fmt.Printf("- id=%d owner=%s balance=%d\n", item.ID, item.Owner, item.Balance)
	}
}
