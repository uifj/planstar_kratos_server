package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ShopCart holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type ShopCart struct {
	ent.Schema
}

// Mixin of the ShopCart.
func (ShopCart) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the ShopCart.
func (ShopCart) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").Default(""),
		field.String("goods_id").Default(""),
		field.String("goods_sn").Default(""),
		field.String("goods_name").Default(""),
		field.String("sku_id").Default(""),
		field.Int64("goods_price").Default(0),
		field.Int32("goods_num").Default(0),
		field.Bool("is_select").Default(true),
		field.Int32("status").Default(1),
	}
}

// Indexes of the ShopCart.
func (ShopCart) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("user_id", "status"),
	}
}

