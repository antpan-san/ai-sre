package models

const (
	CommercialProductTypePack         = "pack"
	CommercialProductTypeTopic        = "topic"
	CommercialProductTypeEnterprise   = "enterprise"
	CommercialProductTypeSingleSkill  = "single_skill"
	CommercialProductStatusActive     = "active"
	CommercialProductStatusDeprecated = "deprecated"

	ProductGrantScopeNode    = "node"
	ProductGrantScopeSubtree = "subtree"
	ProductGrantScopePack    = "pack"
)

// SkillCommercialProduct is a sellable bundle (skillpack.* or pack.*).
type SkillCommercialProduct struct {
	BaseModel
	ProductKey  string `gorm:"size:80;not null;uniqueIndex" json:"product_key"`
	Title       string `gorm:"size:200;not null" json:"title"`
	Description string `gorm:"size:2000" json:"description"`
	ProductType string `gorm:"size:32;not null;default:'pack';index" json:"product_type"`
	Status      string `gorm:"size:32;not null;default:'active';index" json:"status"`
	PriceHint   string `gorm:"size:120" json:"price_hint"`
	SortOrder   int    `gorm:"not null;default:0;index" json:"sort_order"`
}

func (SkillCommercialProduct) TableName() string {
	return "skill_commercial_products"
}

// SkillProductNodeBinding maps a product to skill-tree coordinates.
type SkillProductNodeBinding struct {
	BaseModel
	ProductKey    string `gorm:"size:80;not null;index:idx_product_node_binding" json:"product_key"`
	NodePath      string `gorm:"size:300;index:idx_product_node_binding" json:"node_path"`
	SkillKey      string `gorm:"size:160;index" json:"skill_key"`
	CapabilityKey string `gorm:"size:160;index" json:"capability_key"`
	PackKey       string `gorm:"size:80;index" json:"pack_key"`
	GrantScope    string `gorm:"size:32;not null;default:'subtree'" json:"grant_scope"`
}

func (SkillProductNodeBinding) TableName() string {
	return "skill_product_node_bindings"
}
