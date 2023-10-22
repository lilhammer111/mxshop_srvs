package handler

import (
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
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
	orderListener := OrderListener{}
	p, err := rocketmq.NewTransactionProducer(
		&orderListener,
		producer.WithNameServer([]string{"127.0.0.1:9876"}),
		producer.WithInstanceName("SELL_PRODUCER"))
	if err != nil {
		zap.S().Errorf("failed to generate producer because %s", err.Error())
		return nil, err
	}
	err = p.Start()
	if err != nil {
		zap.S().Errorf("producer failed to start because %s", err.Error())
		return nil, err
	}

	defer func() {
		err = p.Shutdown()
		if err != nil {
			zap.S().Errorf("producer failed to start because %s", err.Error())
		}
	}()

	order := model.OrderInfo{
		OrderSn:      GenerateOrderSn(r.UserId),
		Address:      r.Address,
		SignerName:   r.Name,
		SignerMobile: r.Mobile,
		Post:         r.Post,
		User:         r.UserId,
	}
	jsonString, _ := json.Marshal(order)

	_, err = p.SendMessageInTransaction(context.Background(), primitive.NewMessage("order_reback", jsonString))
	if err != nil {
		return nil, status.Error(codes.Internal, "rocketMQ producer failed to send msg")
	}

	if orderListener.Code != codes.OK {
		return nil, status.Error(orderListener.Code, orderListener.ErrorMessage)
	}

	return &proto.OrderInfoResponse{
		Id:      orderListener.ID,
		OrderSn: order.OrderSn,
		Total:   orderListener.OrderAmount,
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
