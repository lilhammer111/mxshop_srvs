package handler

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"mxshop_srvs/order_srv/global"
	"mxshop_srvs/order_srv/model"
	"mxshop_srvs/order_srv/proto"
)

type OrderServer struct {
	proto.UnimplementedOrderServer
}

func (o OrderServer) CartItemList(ctx context.Context, r *proto.UserInfo) (*proto.CartItemListResponse, error) {
	var shopCarts []model.ShoppingCart
	rsp := new(proto.CartItemListResponse)
	if res := global.DB.Where(&model.ShoppingCart{User: r.Id}).Find(&shopCarts); res.Error != nil {
		return nil, res.Error
	} else {
		rsp.Total = int32(res.RowsAffected)
	}

	for _, shopCart := range shopCarts {
		rsp.Data = append(rsp.Data, &proto.ShopCartInfoResponse{
			Id:      shopCart.ID,
			UserId:  shopCart.User,
			GoodsId: shopCart.Goods,
			Nums:    shopCart.Nums,
			Checked: shopCart.Checked,
		})
	}

	return rsp, nil
}

func (o OrderServer) CreateCartItem(ctx context.Context, r *proto.CartItemRequest) (*proto.ShopCartInfoResponse, error) {
	// scenario1: If the commodity already exists, we need to update the number of the commodity
	// scenario2: If the commodity doesn't exist, we need to create a new one
	var shopCart model.ShoppingCart
	if res := global.DB.Where(&model.ShoppingCart{Goods: r.GoodsId, User: r.UserId}).First(&shopCart); res.RowsAffected == 1 {
		// todo: Could there be concurrency security issues here?
		shopCart.Nums += r.Nums
	} else {
		shopCart.User = r.UserId
		shopCart.Goods = r.GoodsId
		shopCart.Nums = r.Nums
		// Defaults to false
		shopCart.Checked = false
	}
	global.DB.Save(&shopCart)
	return &proto.ShopCartInfoResponse{Id: shopCart.ID}, nil
}

func (o OrderServer) UpdateCartItem(ctx context.Context, r *proto.CartItemRequest) (*emptypb.Empty, error) {
	//
	var shopCart model.ShoppingCart
	if res := global.DB.Where("goods =? and user =?", r.GoodsId, r.UserId).First(&shopCart); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "shopping cart record does not exist")
	}

	shopCart.Checked = r.Checked
	// Attention: If the caller had only passed in the 'Checked' parameter,
	// then the 'Nums' would have been a zero value here, i.e., 0.
	// Blindly assigning the 'Nums' to a value may result in an error that does not follow the business logic.
	// So here we need to make a conditional judgment on the 'Nums'.
	if r.Nums > 0 {
		shopCart.Nums = r.Nums
	}

	global.DB.Save(&shopCart)

	return &emptypb.Empty{}, nil
}

func (o OrderServer) DeleteCartItem(ctx context.Context, r *proto.CartItemRequest) (*emptypb.Empty, error) {
	if res := global.DB.Where("goods =? and user =?", r.GoodsId, r.UserId).Delete(&model.ShoppingCart{}); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "shopping cart record does not exist")
	}
	return &emptypb.Empty{}, nil
}

func (o OrderServer) CreateOrder(ctx context.Context, r *proto.OrderRequest) (*proto.OrderInfoResponse, error) {
	// Attention: Prices of commodities should be queried from the database,so we need to access the commodity service
	// step1: Checking purchased items from the shopping cart
	var shoppingCarts []model.ShoppingCart
	goodsNumsMap := make(map[int32]int32)
	if res := global.DB.Where(model.ShoppingCart{User: r.UserId, Checked: true}).Find(&shoppingCarts); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "there are no items that need to be settled")
	}
	// step2: Query the price of an item from the database
	var goodsIds []int32

	for _, shoppingCart := range shoppingCarts {
		goodsIds = append(goodsIds, shoppingCart.Goods)
		goodsNumsMap[shoppingCart.Goods] = shoppingCart.Nums

		//goodsInvInfos = append(goodsInvInfos, &proto.GoodsInvInfo{
		//	GoodsId: shoppingCart.Goods,
		//	Num:     shoppingCart.Nums,
		//})

	}
	goods, err := global.GoodsSrvClient.BatchGetGoods(context.Background(), &proto.BatchGoodsIdInfo{Id: goodsIds})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query goods information in batch")
	}
	var orderAmount float32
	//var orderGoods []*model.OrderGoods
	var orderGoods []*model.OrderGoods
	var goodsInvInfos []*proto.GoodsInvInfo

	for _, g := range goods.Data {
		orderAmount += g.ShopPrice * float32(goodsNumsMap[g.Id])
		orderGoods = append(orderGoods, &model.OrderGoods{
			Goods:      g.Id,
			GoodsName:  g.Name,
			GoodsImage: g.GoodsFrontImage,
			GoodsPrice: g.ShopPrice,
			Nums:       goodsNumsMap[g.Id],
		})
		goodsInvInfos = append(goodsInvInfos, &proto.GoodsInvInfo{
			GoodsId: g.Id,
			Num:     goodsNumsMap[g.Id],
		})

	}

	// step3: Deduct inventory
	_, err = global.InventorySrvClient.Sell(context.Background(), &proto.SellInfo{GoodsInfo: goodsInvInfos})
	if err != nil {
		return nil, status.Errorf(codes.ResourceExhausted, "failed to deduct inventory")
	}
	// step4: Generate order records
	tx := global.DB.Begin()
	order := model.OrderInfo{
		OrderSn:      GenerateOrderSn(r.UserId),
		OrderMount:   orderAmount,
		PayTime:      nil,
		Address:      r.Address,
		SignerName:   r.Name,
		SignerMobile: r.Mobile,
		Post:         r.Post,
		User:         r.UserId,
	}

	if res := tx.Save(&order); res.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to create table of 'order_info'")
	}
	for _, og := range orderGoods {
		og.Order = order.ID
	}
	// batch insert

	if res := tx.CreateInBatches(orderGoods, 100); res.RowsAffected == 0 || tx.Error != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to create table of 'order_goods'")
	}

	// delete shopping cart
	if res := tx.Where(model.ShoppingCart{User: r.UserId, Checked: true}).Delete(&model.ShoppingCart{}); res.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "failed to delete ordered items in table of 'shoppingCart'")
	}
	tx.Commit()
	return &proto.OrderInfoResponse{
		Id:      order.ID,
		OrderSn: order.OrderSn,
		Total:   order.OrderMount,
	}, nil
}

func (o OrderServer) OrderList(ctx context.Context, r *proto.OrderFilterRequest) (*proto.OrderListResponse, error) {
	var orders []model.OrderInfo
	rsp := new(proto.OrderListResponse)
	var total int64
	if r.UserId == 0 {
		global.DB.Model(&model.OrderInfo{}).Count(&total)
		rsp.Total = int32(total)
	} else {
		global.DB.Where(&model.OrderInfo{User: r.UserId}).Count(&total)
		rsp.Total = int32(total)
	}

	global.DB.Scopes(Paginate(int(r.Pages), int(r.PagePerNums))).Where(&model.OrderInfo{User: r.UserId}).Find(&orders)
	for _, order := range orders {
		rsp.Data = append(rsp.Data, &proto.OrderInfoResponse{
			Id:      order.ID,
			UserId:  order.User,
			OrderSn: order.OrderSn,
			PayType: order.PayType,
			Status:  order.Status,
			Post:    order.Post,
			Total:   order.OrderMount,
			Address: order.Address,
			Name:    order.SignerName,
			Mobile:  order.SignerMobile,
			AddTime: order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return rsp, nil
}

func (o OrderServer) OrderDetail(ctx context.Context, r *proto.OrderRequest) (*proto.OrderInfoDetailResponse, error) {
	var order model.OrderInfo
	rsp := new(proto.OrderInfoDetailResponse)
	if res := global.DB.Where(&model.OrderInfo{
		BaseModel: model.BaseModel{ID: r.Id},
		User:      r.UserId,
	}).First(&order); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "order record does not exist")
	}

	orderInfo := proto.OrderInfoResponse{}
	orderInfo.Id = order.ID
	orderInfo.UserId = order.User
	orderInfo.OrderSn = order.OrderSn
	orderInfo.PayType = order.PayType
	orderInfo.Status = order.Status
	orderInfo.Post = order.Post
	orderInfo.Total = order.OrderMount
	orderInfo.Address = order.Address
	orderInfo.Name = order.SignerName
	orderInfo.Mobile = order.SignerMobile

	rsp.OrderInfo = &orderInfo

	var orderGoods []model.OrderGoods
	if res := global.DB.Where(&model.OrderGoods{Order: order.ID}).Find(&orderGoods); res.Error != nil {
		return nil, res.Error
	}
	for _, orderGood := range orderGoods {
		rsp.Goods = append(rsp.Goods, &proto.OrderItemResponse{
			GoodsId:    orderGood.Goods,
			GoodsName:  orderGood.GoodsName,
			GoodsPrice: orderGood.GoodsPrice,
			GoodsImage: orderGood.GoodsImage,
			Nums:       orderGood.Nums,
		})
	}
	return rsp, nil
}

func (o OrderServer) UpdateOrderStatus(ctx context.Context, r *proto.OrderStatus) (*emptypb.Empty, error) {
	if res := global.DB.Model(&model.OrderInfo{}).Where("order_sn = ?").Update("status", r.Status); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "order record does not exist")
	}
	return &emptypb.Empty{}, nil
}
