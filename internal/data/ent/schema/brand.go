package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Brand holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type Brand struct {
	ent.Schema
}

// Mixin of the Brand.
func (Brand) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the Brand.
func (Brand) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Default(""),
		field.String("logo").Default(""),
		field.Int32("status").Default(1),
	}
}

// Indexes of the Brand.
func (Brand) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
	}
}

