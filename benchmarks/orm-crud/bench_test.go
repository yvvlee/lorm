package ormcrud

import (
	"database/sql"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/yvvlee/lorm"
	entbench "github.com/yvvlee/lorm/benchmarks/orm-crud/ent"
	entuser "github.com/yvvlee/lorm/benchmarks/orm-crud/ent/user"
	"github.com/yvvlee/lorm/builder"
	gmysql "gorm.io/driver/mysql"
	gpostgres "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"xorm.io/xorm"
)

const batchSize = 100

func BenchmarkCreate(b *testing.B) {
	b.Run("lorm", benchmarkCreateLorm)
	b.Run("gorm", benchmarkCreateGorm)
	b.Run("xorm", benchmarkCreateXorm)
	b.Run("ent", benchmarkCreateEnt)
}

func BenchmarkReadByID(b *testing.B) {
	b.Run("lorm", benchmarkReadByIDLorm)
	b.Run("gorm", benchmarkReadByIDGorm)
	b.Run("xorm", benchmarkReadByIDXorm)
	b.Run("ent", benchmarkReadByIDEnt)
}

func BenchmarkUpdateByID(b *testing.B) {
	b.Run("lorm", benchmarkUpdateByIDLorm)
	b.Run("gorm", benchmarkUpdateByIDGorm)
	b.Run("xorm", benchmarkUpdateByIDXorm)
	b.Run("ent", benchmarkUpdateByIDEnt)
}

func BenchmarkDeleteByID(b *testing.B) {
	b.Run("lorm", benchmarkDeleteByIDLorm)
	b.Run("gorm", benchmarkDeleteByIDGorm)
	b.Run("xorm", benchmarkDeleteByIDXorm)
	b.Run("ent", benchmarkDeleteByIDEnt)
}

func BenchmarkBatchCreate100(b *testing.B) {
	b.Run("lorm", benchmarkBatchCreateLorm)
	b.Run("gorm", benchmarkBatchCreateGorm)
	b.Run("xorm", benchmarkBatchCreateXorm)
	b.Run("ent", benchmarkBatchCreateEnt)
}

func BenchmarkBatchRead100(b *testing.B) {
	b.Run("lorm", benchmarkBatchReadLorm)
	b.Run("gorm", benchmarkBatchReadGorm)
	b.Run("xorm", benchmarkBatchReadXorm)
	b.Run("ent", benchmarkBatchReadEnt)
}

func BenchmarkBatchUpdate100(b *testing.B) {
	b.Run("lorm", benchmarkBatchUpdateLorm)
	b.Run("gorm", benchmarkBatchUpdateGorm)
	b.Run("xorm", benchmarkBatchUpdateXorm)
	b.Run("ent", benchmarkBatchUpdateEnt)
}

func BenchmarkBatchDelete100(b *testing.B) {
	b.Run("lorm", benchmarkBatchDeleteLorm)
	b.Run("gorm", benchmarkBatchDeleteGorm)
	b.Run("xorm", benchmarkBatchDeleteXorm)
	b.Run("ent", benchmarkBatchDeleteEnt)
}

func setupLorm(b *testing.B, name string) *lorm.Engine {
	b.Helper()
	db := prepareBenchmarkDatabase(b, "lorm", name)

	engine, err := lorm.NewEngine(
		db.backend.lormDriver,
		db.dsn,
		lorm.WithLogger(noopLogger{}),
		lorm.WithMaxOpenConns(1),
	)
	if err != nil {
		b.Fatalf("open lorm engine: %v", err)
	}
	b.Cleanup(func() {
		_ = engine.Close()
	})
	return engine
}

func setupGorm(b *testing.B, name string) *gorm.DB {
	b.Helper()
	dbConfig := prepareBenchmarkDatabase(b, "gorm", name)

	var dialector gorm.Dialector
	switch dbConfig.backend.name {
	case "sqlite":
		dialector = sqlite.Open(dbConfig.dsn)
	case "mysql":
		dialector = gmysql.Open(dbConfig.dsn)
	case "postgres":
		dialector = gpostgres.Open(dbConfig.dsn)
	default:
		b.Fatalf("unsupported gorm backend %q", dbConfig.backend.name)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	if err != nil {
		b.Fatalf("open gorm db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("get gorm sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	b.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func setupXorm(b *testing.B, name string) *xorm.Engine {
	b.Helper()
	db := prepareBenchmarkDatabase(b, "xorm", name)

	engine, err := xorm.NewEngine(db.backend.sqlDriver, db.dsn)
	if err != nil {
		b.Fatalf("open xorm engine: %v", err)
	}
	engine.SetMaxOpenConns(1)
	b.Cleanup(func() {
		_ = engine.Close()
	})
	return engine
}

func setupEnt(b *testing.B, name string) *entbench.Client {
	b.Helper()
	db := prepareBenchmarkDatabase(b, "ent", name)

	var entDialect string
	switch db.backend.entDialect {
	case "sqlite":
		entDialect = dialect.SQLite
	case "mysql":
		entDialect = dialect.MySQL
	case "postgres":
		entDialect = dialect.Postgres
	default:
		b.Fatalf("unsupported ent backend %q", db.backend.entDialect)
	}

	var (
		drv *entsql.Driver
		err error
	)
	if db.backend.sqlDriver == "pgx" {
		sqlDB, openErr := sql.Open(db.backend.sqlDriver, db.dsn)
		if openErr != nil {
			b.Fatalf("open pgx sql db: %v", openErr)
		}
		drv = entsql.OpenDB(entDialect, sqlDB)
	} else {
		drv, err = entsql.Open(entDialect, db.dsn)
		if err != nil {
			b.Fatalf("open ent driver: %v", err)
		}
	}
	sqlDB := drv.DB()
	sqlDB.SetMaxOpenConns(1)

	client := entbench.NewClient(entbench.Driver(drv))
	b.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

func benchmarkCreateLorm(b *testing.B) {
	engine := setupLorm(b, "create")
	repo := lorm.NewRepository[*LormUser](engine)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &LormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if _, err := repo.Insert(benchmarkCtx, user); err != nil {
			b.Fatalf("lorm create: %v", err)
		}
	}
}

func benchmarkCreateGorm(b *testing.B) {
	db := setupGorm(b, "create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &GormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if err := db.Create(user).Error; err != nil {
			b.Fatalf("gorm create: %v", err)
		}
	}
}

func benchmarkCreateXorm(b *testing.B) {
	engine := setupXorm(b, "create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &XormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if _, err := engine.Insert(user); err != nil {
			b.Fatalf("xorm create: %v", err)
		}
	}
}

func benchmarkCreateEnt(b *testing.B) {
	client := setupEnt(b, "create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		if _, err := client.User.Create().SetName(input.Name).SetAge(input.Age).SetEmail(input.Email).Save(benchmarkCtx); err != nil {
			b.Fatalf("ent create: %v", err)
		}
	}
}

func benchmarkReadByIDLorm(b *testing.B) {
	engine := setupLorm(b, "read")
	repo := lorm.NewRepository[*LormUser](engine)
	seed := makeBenchInput(0)
	user := &LormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if _, err := repo.Insert(benchmarkCtx, user); err != nil {
		b.Fatalf("seed lorm read: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Get(benchmarkCtx, user.ID); err != nil {
			b.Fatalf("lorm read: %v", err)
		}
	}
}

func benchmarkReadByIDGorm(b *testing.B) {
	db := setupGorm(b, "read")
	seed := makeBenchInput(0)
	user := &GormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if err := db.Create(user).Error; err != nil {
		b.Fatalf("seed gorm read: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out GormUser
		if err := db.First(&out, user.ID).Error; err != nil {
			b.Fatalf("gorm read: %v", err)
		}
	}
}

func benchmarkReadByIDXorm(b *testing.B) {
	engine := setupXorm(b, "read")
	seed := makeBenchInput(0)
	user := &XormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if _, err := engine.Insert(user); err != nil {
		b.Fatalf("seed xorm read: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out XormUser
		has, err := engine.ID(user.ID).Get(&out)
		if err != nil {
			b.Fatalf("xorm read: %v", err)
		}
		if !has {
			b.Fatal("xorm read: row not found")
		}
	}
}

func benchmarkReadByIDEnt(b *testing.B) {
	client := setupEnt(b, "read")
	seed := makeBenchInput(0)
	user, err := client.User.Create().SetName(seed.Name).SetAge(seed.Age).SetEmail(seed.Email).Save(benchmarkCtx)
	if err != nil {
		b.Fatalf("seed ent read: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.User.Get(benchmarkCtx, user.ID); err != nil {
			b.Fatalf("ent read: %v", err)
		}
	}
}

func benchmarkUpdateByIDLorm(b *testing.B) {
	engine := setupLorm(b, "update")
	repo := lorm.NewRepository[*LormUser](engine)
	seed := makeBenchInput(0)
	user := &LormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if _, err := repo.Insert(benchmarkCtx, user); err != nil {
		b.Fatalf("seed lorm update: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i + 1)
		if _, err := lorm.Update[*LormUser](engine).
			ID(user.ID).
			SetMap(map[string]any{"name": input.Name, "age": input.Age, "email": input.Email}).
			Exec(benchmarkCtx); err != nil {
			b.Fatalf("lorm update: %v", err)
		}
	}
}

func benchmarkUpdateByIDGorm(b *testing.B) {
	db := setupGorm(b, "update")
	seed := makeBenchInput(0)
	user := &GormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if err := db.Create(user).Error; err != nil {
		b.Fatalf("seed gorm update: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i + 1)
		if err := db.Model(&GormUser{}).Where("id = ?", user.ID).Updates(map[string]any{
			"name":  input.Name,
			"age":   input.Age,
			"email": input.Email,
		}).Error; err != nil {
			b.Fatalf("gorm update: %v", err)
		}
	}
}

func benchmarkUpdateByIDXorm(b *testing.B) {
	engine := setupXorm(b, "update")
	seed := makeBenchInput(0)
	user := &XormUser{Name: seed.Name, Age: seed.Age, Email: seed.Email}
	if _, err := engine.Insert(user); err != nil {
		b.Fatalf("seed xorm update: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i + 1)
		if _, err := engine.ID(user.ID).Cols("name", "age", "email").Update(&XormUser{
			Name:  input.Name,
			Age:   input.Age,
			Email: input.Email,
		}); err != nil {
			b.Fatalf("xorm update: %v", err)
		}
	}
}

func benchmarkUpdateByIDEnt(b *testing.B) {
	client := setupEnt(b, "update")
	seed := makeBenchInput(0)
	user, err := client.User.Create().SetName(seed.Name).SetAge(seed.Age).SetEmail(seed.Email).Save(benchmarkCtx)
	if err != nil {
		b.Fatalf("seed ent update: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i + 1)
		if _, err := client.User.UpdateOneID(user.ID).SetName(input.Name).SetAge(input.Age).SetEmail(input.Email).Save(benchmarkCtx); err != nil {
			b.Fatalf("ent update: %v", err)
		}
	}
}

func benchmarkDeleteByIDLorm(b *testing.B) {
	engine := setupLorm(b, "delete")
	repo := lorm.NewRepository[*LormUser](engine)
	ids := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &LormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if _, err := repo.Insert(benchmarkCtx, user); err != nil {
			b.Fatalf("seed lorm delete: %v", err)
		}
		ids[i] = user.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lorm.DeleteModel[*LormUser](engine).ID(ids[i]).Exec(benchmarkCtx); err != nil {
			b.Fatalf("lorm delete: %v", err)
		}
	}
}

func benchmarkDeleteByIDGorm(b *testing.B) {
	db := setupGorm(b, "delete")
	ids := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &GormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if err := db.Create(user).Error; err != nil {
			b.Fatalf("seed gorm delete: %v", err)
		}
		ids[i] = user.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Delete(&GormUser{}, ids[i]).Error; err != nil {
			b.Fatalf("gorm delete: %v", err)
		}
	}
}

func benchmarkDeleteByIDXorm(b *testing.B) {
	engine := setupXorm(b, "delete")
	ids := make([]int64, b.N)
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user := &XormUser{Name: input.Name, Age: input.Age, Email: input.Email}
		if _, err := engine.Insert(user); err != nil {
			b.Fatalf("seed xorm delete: %v", err)
		}
		ids[i] = user.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.ID(ids[i]).Delete(&XormUser{}); err != nil {
			b.Fatalf("xorm delete: %v", err)
		}
	}
}

func benchmarkDeleteByIDEnt(b *testing.B) {
	client := setupEnt(b, "delete")
	ids := make([]int, b.N)
	for i := 0; i < b.N; i++ {
		input := makeBenchInput(i)
		user, err := client.User.Create().SetName(input.Name).SetAge(input.Age).SetEmail(input.Email).Save(benchmarkCtx)
		if err != nil {
			b.Fatalf("seed ent delete: %v", err)
		}
		ids[i] = user.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.User.DeleteOneID(ids[i]).Exec(benchmarkCtx); err != nil {
			b.Fatalf("ent delete: %v", err)
		}
	}
}

func benchmarkBatchCreateLorm(b *testing.B) {
	engine := setupLorm(b, "batch_create")
	repo := lorm.NewRepository[*LormUser](engine)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := makeLormUsers(i*batchSize, batchSize)
		if _, err := repo.InsertAll(benchmarkCtx, users); err != nil {
			b.Fatalf("lorm batch create: %v", err)
		}
	}
}

func benchmarkBatchCreateGorm(b *testing.B) {
	db := setupGorm(b, "batch_create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := makeGormUsers(i*batchSize, batchSize)
		if err := db.CreateInBatches(users, batchSize).Error; err != nil {
			b.Fatalf("gorm batch create: %v", err)
		}
	}
}

func benchmarkBatchCreateXorm(b *testing.B) {
	engine := setupXorm(b, "batch_create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		users := makeXormUsers(i*batchSize, batchSize)
		if _, err := engine.Insert(&users); err != nil {
			b.Fatalf("xorm batch create: %v", err)
		}
	}
}

func benchmarkBatchCreateEnt(b *testing.B) {
	client := setupEnt(b, "batch_create")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builders := makeEntCreateBuilders(client, i*batchSize, batchSize)
		if _, err := client.User.CreateBulk(builders...).Save(benchmarkCtx); err != nil {
			b.Fatalf("ent batch create: %v", err)
		}
	}
}

func benchmarkBatchReadLorm(b *testing.B) {
	engine := setupLorm(b, "batch_read")
	repo := lorm.NewRepository[*LormUser](engine)
	users := makeLormUsers(0, batchSize)
	if _, err := repo.InsertAll(benchmarkCtx, users); err != nil {
		b.Fatalf("seed lorm batch read: %v", err)
	}
	emails := lormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := lorm.Query[*LormUser](engine).
			Where(builder.In("email", emails)).
			Find(benchmarkCtx)
		if err != nil {
			b.Fatalf("lorm batch read: %v", err)
		}
		if len(out) != batchSize {
			b.Fatalf("lorm batch read: got %d rows", len(out))
		}
	}
}

func benchmarkBatchReadGorm(b *testing.B) {
	db := setupGorm(b, "batch_read")
	users := makeGormUsers(0, batchSize)
	if err := db.CreateInBatches(users, batchSize).Error; err != nil {
		b.Fatalf("seed gorm batch read: %v", err)
	}
	emails := gormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []GormUser
		if err := db.Where("email IN ?", emails).Find(&out).Error; err != nil {
			b.Fatalf("gorm batch read: %v", err)
		}
		if len(out) != batchSize {
			b.Fatalf("gorm batch read: got %d rows", len(out))
		}
	}
}

func benchmarkBatchReadXorm(b *testing.B) {
	engine := setupXorm(b, "batch_read")
	users := makeXormUsers(0, batchSize)
	if _, err := engine.Insert(&users); err != nil {
		b.Fatalf("seed xorm batch read: %v", err)
	}
	emails := xormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []XormUser
		if err := engine.In("email", emails).Find(&out); err != nil {
			b.Fatalf("xorm batch read: %v", err)
		}
		if len(out) != batchSize {
			b.Fatalf("xorm batch read: got %d rows", len(out))
		}
	}
}

func benchmarkBatchReadEnt(b *testing.B) {
	client := setupEnt(b, "batch_read")
	created, err := client.User.CreateBulk(makeEntCreateBuilders(client, 0, batchSize)...).Save(benchmarkCtx)
	if err != nil {
		b.Fatalf("seed ent batch read: %v", err)
	}
	emails := entEmails(created)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := client.User.Query().Where(entuser.EmailIn(emails...)).All(benchmarkCtx)
		if err != nil {
			b.Fatalf("ent batch read: %v", err)
		}
		if len(out) != batchSize {
			b.Fatalf("ent batch read: got %d rows", len(out))
		}
	}
}

func benchmarkBatchUpdateLorm(b *testing.B) {
	engine := setupLorm(b, "batch_update")
	repo := lorm.NewRepository[*LormUser](engine)
	users := makeLormUsers(0, batchSize)
	if _, err := repo.InsertAll(benchmarkCtx, users); err != nil {
		b.Fatalf("seed lorm batch update: %v", err)
	}
	emails := lormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := lorm.Update[*LormUser](engine).
			Where(builder.In("email", emails)).
			SetMap(map[string]any{"age": 100 + i}).
			Exec(benchmarkCtx); err != nil {
			b.Fatalf("lorm batch update: %v", err)
		}
	}
}

func benchmarkBatchUpdateGorm(b *testing.B) {
	db := setupGorm(b, "batch_update")
	users := makeGormUsers(0, batchSize)
	if err := db.CreateInBatches(users, batchSize).Error; err != nil {
		b.Fatalf("seed gorm batch update: %v", err)
	}
	emails := gormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := db.Model(&GormUser{}).Where("email IN ?", emails).Updates(map[string]any{"age": 100 + i}).Error; err != nil {
			b.Fatalf("gorm batch update: %v", err)
		}
	}
}

func benchmarkBatchUpdateXorm(b *testing.B) {
	engine := setupXorm(b, "batch_update")
	users := makeXormUsers(0, batchSize)
	if _, err := engine.Insert(&users); err != nil {
		b.Fatalf("seed xorm batch update: %v", err)
	}
	emails := xormEmails(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Table("bench_users").In("email", emails).Update(map[string]any{"age": 100 + i}); err != nil {
			b.Fatalf("xorm batch update: %v", err)
		}
	}
}

func benchmarkBatchUpdateEnt(b *testing.B) {
	client := setupEnt(b, "batch_update")
	created, err := client.User.CreateBulk(makeEntCreateBuilders(client, 0, batchSize)...).Save(benchmarkCtx)
	if err != nil {
		b.Fatalf("seed ent batch update: %v", err)
	}
	emails := entEmails(created)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.User.Update().Where(entuser.EmailIn(emails...)).SetAge(100 + i).Save(benchmarkCtx); err != nil {
			b.Fatalf("ent batch update: %v", err)
		}
	}
}

func benchmarkBatchDeleteLorm(b *testing.B) {
	engine := setupLorm(b, "batch_delete")
	repo := lorm.NewRepository[*LormUser](engine)
	total := b.N * batchSize
	seed := makeLormUsers(0, total)
	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}
		if _, err := repo.InsertAll(benchmarkCtx, seed[start:end]); err != nil {
			b.Fatalf("seed lorm batch delete: %v", err)
		}
	}
	emails := lormEmails(seed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchEmails := emails[i*batchSize : (i+1)*batchSize]
		if _, err := lorm.DeleteModel[*LormUser](engine).
			Where(builder.In("email", batchEmails)).
			Exec(benchmarkCtx); err != nil {
			b.Fatalf("lorm batch delete: %v", err)
		}
	}
}

func benchmarkBatchDeleteGorm(b *testing.B) {
	db := setupGorm(b, "batch_delete")
	total := b.N * batchSize
	seed := makeGormUsers(0, total)
	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}
		if err := db.CreateInBatches(seed[start:end], batchSize).Error; err != nil {
			b.Fatalf("seed gorm batch delete: %v", err)
		}
	}
	emails := gormEmails(seed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchEmails := emails[i*batchSize : (i+1)*batchSize]
		if err := db.Where("email IN ?", batchEmails).Delete(&GormUser{}).Error; err != nil {
			b.Fatalf("gorm batch delete: %v", err)
		}
	}
}

func benchmarkBatchDeleteXorm(b *testing.B) {
	engine := setupXorm(b, "batch_delete")
	total := b.N * batchSize
	seed := makeXormUsers(0, total)
	for start := 0; start < total; start += batchSize {
		end := start + batchSize
		if end > total {
			end = total
		}
		chunk := seed[start:end]
		if _, err := engine.Insert(&chunk); err != nil {
			b.Fatalf("seed xorm batch delete: %v", err)
		}
	}
	emails := xormEmails(seed)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchEmails := emails[i*batchSize : (i+1)*batchSize]
		if _, err := engine.In("email", batchEmails).Delete(&XormUser{}); err != nil {
			b.Fatalf("xorm batch delete: %v", err)
		}
	}
}

func benchmarkBatchDeleteEnt(b *testing.B) {
	client := setupEnt(b, "batch_delete")
	total := b.N * batchSize
	all := make([]*entbench.User, 0, total)
	for start := 0; start < total; start += batchSize {
		size := batchSize
		if remain := total - start; remain < size {
			size = remain
		}
		chunk, err := client.User.CreateBulk(makeEntCreateBuilders(client, start, size)...).Save(benchmarkCtx)
		if err != nil {
			b.Fatalf("seed ent batch delete: %v", err)
		}
		all = append(all, chunk...)
	}
	emails := entEmails(all)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchEmails := emails[i*batchSize : (i+1)*batchSize]
		if _, err := client.User.Delete().Where(entuser.EmailIn(batchEmails...)).Exec(benchmarkCtx); err != nil {
			b.Fatalf("ent batch delete: %v", err)
		}
	}
}

func makeLormUsers(offset, size int) []*LormUser {
	users := make([]*LormUser, size)
	for i := 0; i < size; i++ {
		input := makeBenchInput(offset + i)
		users[i] = &LormUser{Name: input.Name, Age: input.Age, Email: input.Email}
	}
	return users
}

func makeGormUsers(offset, size int) []GormUser {
	users := make([]GormUser, size)
	for i := 0; i < size; i++ {
		input := makeBenchInput(offset + i)
		users[i] = GormUser{Name: input.Name, Age: input.Age, Email: input.Email}
	}
	return users
}

func makeXormUsers(offset, size int) []XormUser {
	users := make([]XormUser, size)
	for i := 0; i < size; i++ {
		input := makeBenchInput(offset + i)
		users[i] = XormUser{Name: input.Name, Age: input.Age, Email: input.Email}
	}
	return users
}

func makeEntCreateBuilders(client *entbench.Client, offset, size int) []*entbench.UserCreate {
	builders := make([]*entbench.UserCreate, size)
	for i := 0; i < size; i++ {
		input := makeBenchInput(offset + i)
		builders[i] = client.User.Create().SetName(input.Name).SetAge(input.Age).SetEmail(input.Email)
	}
	return builders
}

func lormIDs(users []*LormUser) []int64 {
	ids := make([]int64, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func lormEmails(users []*LormUser) []string {
	emails := make([]string, len(users))
	for i, user := range users {
		emails[i] = user.Email
	}
	return emails
}

func gormIDs(users []GormUser) []int64 {
	ids := make([]int64, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func gormEmails(users []GormUser) []string {
	emails := make([]string, len(users))
	for i, user := range users {
		emails[i] = user.Email
	}
	return emails
}

func xormIDs(users []XormUser) []int64 {
	ids := make([]int64, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func xormEmails(users []XormUser) []string {
	emails := make([]string, len(users))
	for i, user := range users {
		emails[i] = user.Email
	}
	return emails
}

func entIDs(users []*entbench.User) []int {
	ids := make([]int, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func entEmails(users []*entbench.User) []string {
	emails := make([]string, len(users))
	for i, user := range users {
		emails[i] = user.Email
	}
	return emails
}
