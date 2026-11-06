package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Goods holds the schema definition for a mall entity.
// All public ids come from IDMixin (UUIDv7); reference columns are plain
// strings to match the "no foreign keys" design of the source project.
type Goods struct {
	ent.Schema
}

// Mixin of the Goods.
func (Goods) Mixin() []ent.Mixin {
	return []ent.Mixin{IDMixin{}, TimeMixin{}}
}

// Fields of the Goods.
func (Goods) Fields() []ent.Field {
	return []ent.Field{
		field.String("category_id").Default(""),
		field.String("brand_id").Default(""),
		field.String("type_id").Default(""),
		field.String("name").Default(""),
		field.String("name_alias").Default(""),
		field.String("goods_sn").Unique(),
		field.Text("goods_brief").Default(""),
		field.String("goods_front_image").Default(""),
		field.Int64("market_price").Default(0),
		field.Bool("on_sale").Default(false),
		field.Bool("ship_free").Default(false),
		field.Bool("is_new").Default(false),
		field.Bool("is_hot").Default(false),
		field.Int64("sold_num").Default(0),
		field.Int32("status").Default(1),
	}
}

// Indexes of the Goods.
func (Goods) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("category_id"),
		index.Fields("brand_id"),
		index.Fields("status"),
	}
}

