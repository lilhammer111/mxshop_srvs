package handler

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"mxshop_srvs/inventory_srv/global"
	"mxshop_srvs/inventory_srv/model"
	"mxshop_srvs/inventory_srv/proto"
)

type InventoryServer struct {
	proto.UnimplementedInventoryServer
}

func (i InventoryServer) SetInv(c context.Context, r *proto.GoodsInvInfo) (*emptypb.Empty, error) {
	var inv model.Inventory
	global.DB.Where("goods = ?", r.GoodsId).First(&inv)
	// if goodsID's value is 0, it means that the goods is not found
	if inv.Goods == 0 {
		inv.Goods = r.GoodsId
	}
	inv.Stocks = r.Num
	global.DB.Save(&inv)
	return &emptypb.Empty{}, nil
}

func (i InventoryServer) InvDetail(c context.Context, r *proto.GoodsInvInfo) (*proto.GoodsInvInfo, error) {
	var inv model.Inventory
	res := global.DB.Where("goods = ?", r.GoodsId).First(&inv)
	if res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "no inventory information")
	}
	return &proto.GoodsInvInfo{
		GoodsId: inv.Goods,
		Num:     inv.Stocks,
	}, nil
}

func (i InventoryServer) Sell(c context.Context, r *proto.SellInfo) (*emptypb.Empty, error) {
	// todo database transactions
	// todo distributed lock
	tx := global.DB.Begin()

	sellDetail := model.StockSellDetail{
		OrderSn: r.OrderSn,
		Status:  1,
	}

	var details []model.GoodsDetail

	for _, goodsInfo := range r.GoodsInfo {
		details = append(details, model.GoodsDetail{
			Goods: goodsInfo.GoodsId,
			Num:   goodsInfo.Num,
		})

		var inv model.Inventory
		if res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("goods = ?", goodsInfo.GoodsId).First(&inv); res.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "no inventory information")
		}
		// Determining the adequacy of inventory
		if inv.Stocks < goodsInfo.Num {
			tx.Rollback()
			return nil, status.Errorf(codes.ResourceExhausted, "inventory is insufficient")
		}
		inv.Stocks -= goodsInfo.Num
		tx.Save(&inv)
	}

	sellDetail.Detail = details
	if res := tx.Create(&sellDetail); res.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to save deduction inventory history")
	}

	tx.Commit()

	return &emptypb.Empty{}, nil
}

//func (i InventoryServer) Reback(c context.Context, r *proto.SellInfo) (*emptypb.Empty, error) {
//	// There are several possible reasons for the return of stockpiles
//	// scenario 1: The order timed out
//	// scenario 2: The order record failed to be saved
//	// scenario 3: User-initiated order cancellation
//	tx := global.DB.Begin()
//	for _, goodsInfo := range r.GoodsInfo {
//		var inv model.Inventory
//		if res := global.DB.Where("goods = ?", goodsInfo.GoodsId).First(&inv); res.RowsAffected == 0 {
//			tx.Rollback()
//			return nil, status.Errorf(codes.InvalidArgument, "no inventory information")
//		}
//
//		inv.Stocks += goodsInfo.Num
//		tx.Save(&inv)
//	}
//	tx.Commit()
//
//	return &emptypb.Empty{}, nil
//
//}

func AutoReback(c context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	type OrderInfo struct {
		OrderSn string
	}

	for i := range msgs {
		var orderInfo OrderInfo
		err := json.Unmarshal(msgs[i].Body, &orderInfo)
		if err != nil {
			zap.S().Errorf("failed to unmarshal json because %s\n", msgs[i].Body)
			return consumer.ConsumeSuccess, nil
		}

		tx := global.DB.Begin()
		var sellDetail model.StockSellDetail
		if res := tx.Model(&model.StockSellDetail{}).Where(&model.StockSellDetail{OrderSn: orderInfo.OrderSn, Status: 1}).First(&sellDetail); res.RowsAffected == 0 {
			return consumer.ConsumeSuccess, nil
		}
		//如果查询到那么逐个归还库存
		for _, orderGood := range sellDetail.Detail {
			//update怎么用
			//先查询一下inventory表在， update语句的 update xx set stocks=stocks+2
			if result := tx.Model(&model.Inventory{}).Where(&model.Inventory{Goods: orderGood.Goods}).Update("stocks", gorm.Expr("stocks+?", orderGood.Num)); result.RowsAffected == 0 {
				tx.Rollback()
				return consumer.ConsumeRetryLater, nil
			}
		}

		if result := tx.Model(&model.StockSellDetail{}).Where(&model.StockSellDetail{OrderSn: orderInfo.OrderSn}).Update("status", 2); result.RowsAffected == 0 {
			tx.Rollback()
			return consumer.ConsumeRetryLater, nil
		}
		tx.Commit()
		return consumer.ConsumeSuccess, nil
	}
	return consumer.ConsumeSuccess, nil
}
