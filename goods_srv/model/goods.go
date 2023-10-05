package model

type Category struct {
	BaseModel
	Name             string      `gorm:"type:varchar(20);not null" json:"name,omitempty"`
	ParentCategoryID int32       `json:"-"`
	ParentCategory   *Category   `json:"parent-category,omitempty"`
	SubCategory      []*Category `gorm:"foreignKey:ParentCategoryID;references:ID" json:"sub-category,omitempty"`
	Level            int32       `gorm:"type:int;not null;default:1" json:"level,omitempty"`
	IsTab            bool        `gorm:"default:false;not null" json:"is-tab,omitempty"`
}

type Brand struct {
	BaseModel
	Name string `gorm:"type:varchar(20);not null" json:"name,omitempty"`
	Logo string `gorm:"type:varchar(20);default:'';not null" json:"logo,omitempty"`
}

type GoodsCategoryBrand struct {
	BaseModel
	CategoryID int32    `gorm:"type:int;index:idx_category_brand,unique" json:"category-id,omitempty"`
	Category   Category `json:"category"`

	BrandID int32 `gorm:"type:int;index:idx_category_brand,unique" json:"brand-id,omitempty"`
	Brand   Brand `json:"brand"`
}

type Banner struct {
	BaseModel
	Image string `gorm:"type:varchar(200);not null" json:"image,omitempty"`
	URL   string `gorm:"type:varchar(200);not null" json:"url,omitempty"`
	Index int32  `gorm:"type:int;default:1;not null" json:"index,omitempty"`
}

type Good struct {
	BaseModel

	CategoryID int32    `gorm:"type:int;not null" json:"category-id,omitempty"`
	Category   Category `json:"category"`
	BrandID    int32    `gorm:"type:int;not null" json:"brand-id,omitempty"`
	Brand      Brand    `json:"brand"`

	OnSale   bool `gorm:"default:false;not null" json:"on-sale,omitempty"`
	ShipFree bool `gorm:"default:false;not null" json:"ship-free,omitempty"`
	IsNew    bool `gorm:"default:false;not null" json:"is-new,omitempty"`
	IsHot    bool `gorm:"default:false;not null" json:"is-hot,omitempty"`

	Name    string `gorm:"type:varchar(50);not null" json:"name,omitempty"`
	GoodsSn string `gorm:"type:varchar(50);not null" json:"goods-sn,omitempty"`

	HitNum       int32    `gorm:"type:int;default:0;not null" json:"hit-num,omitempty"`
	SalesVolumes int32    `gorm:"type:int;default:0;not null" json:"sales-volumes,omitempty"`
	Favorites    int32    `gorm:"type:int;default:0;not null" json:"favorites,omitempty"`
	MarketPrice  float32  `gorm:"not null" json:"market-price,omitempty"`
	ShopPrice    float32  `gorm:"not null" json:"shop-price,omitempty"`
	Brief        string   `gorm:"type:varchar(50);not null" json:"brief,omitempty"`
	Images       GormList `gorm:"type:varchar(1000);not null" json:"images,omitempty"`
	DescImages   GormList `gorm:"type:varchar(1000);not null" json:"desc-images,omitempty"`
	CoverImage   string   `gorm:"type:varchar(200);not null" json:"cover-image,omitempty"`
}
