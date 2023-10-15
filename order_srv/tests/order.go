package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"mxshop_srvs/order_srv/proto"
)

var orderClient proto.OrderClient
var conn *grpc.ClientConn
var ctx = context.Background()

func Init() {
	var err error
	conn, err = grpc.Dial("127.0.0.1:50054", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	orderClient = proto.NewOrderClient(conn)
}

func main() {
	Init()
	defer conn.Close()

	//err := TestCreateCartItem(1, 422, 1)
	//err = TestCartItemList(1)

	//TestUpdateCartItem(1)

	//err := TestCreateOrder()
	//
	//if err != nil {
	//	fmt.Println(err.Error())
	//}

	//TestGetOrderDetail(20)
	TestOrderList()
}

func TestCreateCartItem(userID, goodsID, nums int32) error {
	rsp, err := orderClient.CreateCartItem(ctx, &proto.CartItemRequest{
		UserId:  userID,
		GoodsId: goodsID,
		Nums:    nums,
	})
	fmt.Println(rsp.Id)

	return err
}

func TestCartItemList(userID int32) error {
	rsp, err := orderClient.CartItemList(ctx, &proto.UserInfo{Id: userID})
	for _, item := range rsp.Data {
		fmt.Println(item.Id)
		fmt.Println(item.GoodsId)
		fmt.Println(item.Nums)
	}
	return err
}

func TestUpdateCartItem(id int32) {
	_, err := orderClient.UpdateCartItem(context.Background(), &proto.CartItemRequest{
		Id:      id,
		Checked: true,
	})
	if err != nil {
		panic(err)
	}
}

func TestCreateOrder() error {
	_, err := orderClient.CreateOrder(context.Background(), &proto.OrderRequest{
		UserId:  1,
		Address: "北京市",
		Name:    "bobby",
		Mobile:  "18787878787",
		Post:    "请尽快发货",
	})
	return err
}

func TestGetOrderDetail(orderID int32) {
	rsp, err := orderClient.OrderDetail(context.Background(), &proto.OrderRequest{Id: orderID})
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp.OrderInfo.OrderSn)
	for _, g := range rsp.Goods {
		fmt.Println(g.GoodsName)
	}
}

func TestOrderList() {
	rsp, err := orderClient.OrderList(context.Background(), &proto.OrderFilterRequest{UserId: 1})
	if err != nil {
		panic(err)
	}
	for _, order := range rsp.Data {
		fmt.Println(order.OrderSn)
	}
}
