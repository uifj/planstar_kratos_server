package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// Record supplies numeric identities and audit fields. IDs are database allocated.
type Record struct{ mixin.Schema }

func (Record) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// User is the shared shop account credential store.
type User struct{ ent.Schema }

func (User) Mixin() []ent.Mixin { return []ent.Mixin{Record{}} }
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("mobile").Unique(),
		field.String("password").Default(""),
		field.String("nick_name").Default(""),
		field.String("gender").Default("male"),
		field.Int("role").Default(1),
		field.Time("birthday").Optional().Nillable(),
		field.Bool("deleted").Default(false),
	}
}
func (User) Indexes() []ent.Index { return []ent.Index{index.Fields("deleted")} }

// Address is a shop user's shipping address.
type Address struct{ ent.Schema }

func (Address) Mixin() []ent.Mixin { return []ent.Mixin{Record{}} }
func (Address) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Int("is_default").Default(0),
		field.String("mobile").Default(""),
		field.String("name").Default(""),
		field.String("province").Default(""),
		field.String("city").Default(""),
		field.String("districts").Default(""),
		field.String("address").Default(""),
		field.String("post_code").Default(""),
		field.Bool("deleted").Default(false),
	}
}
func (Address) Indexes() []ent.Index { return []ent.Index{index.Fields("user_id", "deleted")} }
