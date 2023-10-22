package model

import (
	"database/sql/driver"
	"encoding/json"
)

type Inventory struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"` // goods ID
	Stocks  int32 `gorm:"type:int"`
	Version int32 `gorm:"type:int"` // Optimistic locks for distributed locks
}

type Delivery struct {
	Goods   int32  `gorm:"type:int;index"`
	Nums    int32  `gorm:"type:int"`
	OrderSn string `gorm:"type:varchar(200)"`
	Status  string `gorm:"type:varchar(200)"` //1. 表示等待支付 2. 表示支付成功 3. 失败
}

type GoodsDetail struct {
	Goods int32
	Num   int32
}

type GoodsDetailList []GoodsDetail

func (g GoodsDetailList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// Scan 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (g *GoodsDetailList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), g)
}

type StockSellDetail struct {
	OrderSn string          `gorm:"type:varchar(200);index:idx_order_sn,unique;"`
	Status  int32           `gorm:"type:varchar(200)"` //1 表示已扣减 2. 表示已归还
	Detail  GoodsDetailList `gorm:"type:varchar(200)"`
}

func (StockSellDetail) TableName() string {
	return "stockselldetail"
}

//type InventoryHistory struct {
//	user   int32 `gorm:"type:int"`
//	goods  int32 `gorm:"type:int"`
//	nums   int32 `gorm:"type:int"`
//	order  int32 `gorm:"type:int"`
//	status int32 `gorm:"type:int;comment:1 means pre-deduct,2 means paid"`
//}
