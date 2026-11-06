package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// GoodsSku holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type GoodsSku struct {
	ent.Schema
}

// Mixin of the GoodsSku.
func (GoodsSku) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the GoodsSku.
func (GoodsSku) Fields() []ent.Field {
	return []ent.Field{
		field.String("goods_id").Default(""),
		field.String("sku_name").Default(""),
		field.String("sku_code").Unique(),
		field.String("bar_code").Default(""),
		field.Int64("price").Default(0),
		field.Int64("promotion_price").Default(0),
		field.Int64("inventory").Default(0),
		field.Bool("on_sale").Default(false),
		field.String("pic").Default(""),
		field.JSON("specs", map[string]any{}).Optional(),
		field.Int32("status").Default(1),
	}
}

// Indexes of the GoodsSku.
func (GoodsSku) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("goods_id"),
	}
}

