package handler

import (
	"fmt"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}

		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func GenerateOrderSn(userId int32) string {
	// order generation rule: year month day hour minute second + userID + two-digit random number
	now := time.Now()
	orderSn := fmt.Sprintf(
		"%d%d%d%d%d%d%d%d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		userId,
		rand.Intn(90)+10,
	)
	return orderSn
}
