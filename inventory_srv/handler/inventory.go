package handler

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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
	for _, goodsInfo := range r.GoodsInfo {
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
	tx.Commit()

	return &emptypb.Empty{}, nil
}

func (i InventoryServer) Reback(c context.Context, r *proto.SellInfo) (*emptypb.Empty, error) {
	// There are several possible reasons for the return of stockpiles
	// scenario 1: The order timed out
	// scenario 2: The order record failed to be saved
	// scenario 3: User-initiated order cancellation
	tx := global.DB.Begin()
	for _, goodsInfo := range r.GoodsInfo {
		var inv model.Inventory
		if res := global.DB.Where("goods = ?", goodsInfo.GoodsId).First(&inv); res.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "no inventory information")
		}

		inv.Stocks += goodsInfo.Num
		tx.Save(&inv)
	}
	tx.Commit()

	return &emptypb.Empty{}, nil

}
