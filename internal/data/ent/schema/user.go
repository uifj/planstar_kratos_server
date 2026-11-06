package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type User struct {
	ent.Schema
}

// Mixin of the User.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("mobile").Unique(),
		field.String("username").Default(""),
		field.String("password_hash").Default(""),
		field.String("nick_name").Default(""),
		field.Int64("birthday").Default(0),
		field.String("gender").Default(""),
		field.Int32("role").Default(1),
		field.Int32("status").Default(1),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("role"),
		index.Fields("status"),
	}
}

