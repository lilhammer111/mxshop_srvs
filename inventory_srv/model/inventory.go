package model

type Inventory struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"` // goods ID
	Stocks  int32 `gorm:"type:int"`
	Version int32 `gorm:"type:int"` // Optimistic locks for distributed locks
}

//type InventoryHistory struct {
//	user   int32 `gorm:"type:int"`
//	goods  int32 `gorm:"type:int"`
//	nums   int32 `gorm:"type:int"`
//	order  int32 `gorm:"type:int"`
//	status int32 `gorm:"type:int;comment:1 means pre-deduct,2 means paid"`
//}
