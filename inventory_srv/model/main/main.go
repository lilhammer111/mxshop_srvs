package main

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"mxshop_srvs/inventory_srv/model"
	"os"
	"time"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/mxshop_inventory_srv?charset=utf8mb4&parseTime=True&loc=Local"

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,        // Don't include params in the SQL log
			Colorful:                  true,        // Enable color
		},
	)

	// Globally mode
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         newLogger,
	})
	if err != nil {
		log.Fatal(err)
	}

	//err = db.AutoMigrate(&model.Inventory{}, &model.StockSellDetail{})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//od := model.StockSellDetail{
	//	OrderSn: "imooc-bobby",
	//	Status:  1,
	//	Detail:  []model.GoodsDetail{{1, 2}, {2, 6}},
	//}
	//db.Create(&od)

	var ssd model.StockSellDetail
	db.Where(model.StockSellDetail{OrderSn: "imooc-bobby"}).First(&ssd)
	fmt.Println(ssd.Detail)
}
