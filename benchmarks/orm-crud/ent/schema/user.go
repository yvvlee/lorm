package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	benchmodel "github.com/yvvlee/lorm/benchmarks/orm-crud/benchmodel"
)

type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "bench_users"},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("alias").Optional().Nillable(),
		field.Int("age"),
		field.Int("age_p").Optional().Nillable(),
		field.Bool("active"),
		field.Bool("active_p").Optional().Nillable(),
		field.String("email").Unique(),
		field.JSON("tags", benchmodel.IntSlice{}),
		field.JSON("meta", benchmodel.StringMap{}),
		field.JSON("profile", benchmodel.Profile{}),
		field.JSON("contacts", benchmodel.ContactList{}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name"),
		index.Fields("email").Unique(),
	}
}
