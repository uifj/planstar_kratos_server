package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserAddress holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type UserAddress struct {
	ent.Schema
}

// Mixin of the UserAddress.
func (UserAddress) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the UserAddress.
func (UserAddress) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Default(""),
		field.String("name").Default(""),
		field.String("mobile").Default(""),
		field.String("province").Default(""),
		field.String("city").Default(""),
		field.String("districts").Default(""),
		field.String("address").Default(""),
		field.String("post_code").Default(""),
		field.Int32("is_default").Default(0),
		field.Int32("status").Default(1),
	}
}

// Indexes of the UserAddress.
func (UserAddress) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "status"),
	}
}

