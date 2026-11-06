package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GoodsImage holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type GoodsImage struct {
	ent.Schema
}

// Mixin of the GoodsImage.
func (GoodsImage) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the GoodsImage.
func (GoodsImage) Fields() []ent.Field {
	return []ent.Field{
		field.String("goods_id").Default(""),
		field.String("url").Default(""),
		field.Int32("position").Default(0),
		field.Bool("is_master").Default(false),
		field.Int32("status").Default(1),
	}
}

// Indexes of the GoodsImage.
func (GoodsImage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("goods_id"),
	}
}

