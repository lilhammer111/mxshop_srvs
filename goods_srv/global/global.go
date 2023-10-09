package global

import (
	"gorm.io/gorm"
	"mxshop_srvs/goods_srv/config"
)

var (
	DB           *gorm.DB
	ServerConfig = &config.ServerConfig{}
	NacosConfig  config.NacosConfig
)

//func init() {
//	dsn := "root:root@tcp(127.0.0.1:3306)/mxshop_goods_srv?charset=utf8mb4&parseTime=True&loc=Local"
//
//	newLogger := logger.New(
//		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
//		logger.Config{
//			SlowThreshold:             time.Second, // Slow SQL threshold
//			LogLevel:                  logger.Info, // Log level
//			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
//			ParameterizedQueries:      true,        // Don't include params in the SQL log
//			Colorful:                  true,        // Enable color
//		},
//	)
//
//	// Globally mode
//	var err error
//	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
//		NamingStrategy: schema.NamingStrategy{SingularTable: true},
//		Logger:         newLogger,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//}
