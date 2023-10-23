package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"mxshop_srvs/order_srv/global"
	"mxshop_srvs/order_srv/model"
	"mxshop_srvs/order_srv/proto"
	"time"
)

type OrderListener struct {
	Code         codes.Code
	ErrorMessage string
	ID           int32
	OrderAmount  float32
	Ctx          context.Context
}

func (o *OrderListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	var orderInfo model.OrderInfo
	_ = json.Unmarshal(msg.Body, &orderInfo)

	parentSpan := opentracing.SpanFromContext(o.Ctx)

	// Attention: Prices of commodities should be queried from the database,so we need to access the commodity service
	// Checking purchased items from the shopping cart
	shopCartSpan := opentracing.GlobalTracer().StartSpan("select_shopcart", opentracing.ChildOf(parentSpan.Context()))

	var shoppingCarts []model.ShoppingCart
	goodsNumsMap := make(map[int32]int32)
	if res := global.DB.Where(model.ShoppingCart{User: orderInfo.User, Checked: true}).Find(&shoppingCarts); res.RowsAffected == 0 {
		o.Code = codes.InvalidArgument
		o.ErrorMessage = "there are no items that need to be settled"
		// If an error occurs before inventory has been deducted, then the message should be rolled back
		return primitive.RollbackMessageState
	}
	shopCartSpan.Finish()

	// Query the price of an item from the database
	var goodsIds []int32

	for _, shoppingCart := range shoppingCarts {
		goodsIds = append(goodsIds, shoppingCart.Goods)
		goodsNumsMap[shoppingCart.Goods] = shoppingCart.Nums
	}

	queryGoodsSpan := opentracing.GlobalTracer().StartSpan("query_goods", opentracing.ChildOf(parentSpan.Context()))

	goods, err := global.GoodsSrvClient.BatchGetGoods(context.Background(), &proto.BatchGoodsIdInfo{Id: goodsIds})
	if err != nil {
		o.Code = codes.Internal
		o.ErrorMessage = "failed to query goods information in batch"
		return primitive.RollbackMessageState
	}

	queryGoodsSpan.Finish()

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

	// Deduct inventory
	queryInvSpan := opentracing.GlobalTracer().StartSpan("query_inv", opentracing.ChildOf(parentSpan.Context()))

	_, err = global.InventorySrvClient.Sell(context.Background(), &proto.SellInfo{OrderSn: orderInfo.OrderSn, GoodsInfo: goodsInvInfos})
	if err != nil {
		//todo 如果是网络问题，err.codes 应该是非internal InvalidArgument 或者ResourceExhausted，需要改写一下sell接口的逻辑
		o.Code = codes.ResourceExhausted
		o.ErrorMessage = "failed to deduct inventory"
		return primitive.RollbackMessageState
		//return nil, status.Errorf(codes.ResourceExhausted, "failed to deduct inventory")
	}
	queryInvSpan.Finish()

	// step4: Generate order records
	tx := global.DB.Begin()
	orderInfo.OrderMount = orderAmount

	saveOrderSpan := opentracing.GlobalTracer().StartSpan("save_order", opentracing.ChildOf(parentSpan.Context()))

	if res := tx.Save(&orderInfo); res.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.ErrorMessage = "failed to create order info record"
		// If an error occurs after inventory has not yet been deducted,
		// the message should be submitted for the consumer to perform a restoration of the inventory quantity
		return primitive.CommitMessageState
	}

	saveOrderSpan.Finish()

	o.ID = orderInfo.ID
	o.OrderAmount = orderInfo.OrderMount
	for _, og := range orderGoods {
		og.Order = orderInfo.ID
	}

	// batch insert
	saveOrderGoodsSpan := opentracing.GlobalTracer().StartSpan("save_order_goods", opentracing.ChildOf(parentSpan.Context()))

	if res := tx.CreateInBatches(orderGoods, 100); res.RowsAffected == 0 || tx.Error != nil {
		tx.Rollback()
		o.Code = codes.Internal
		o.ErrorMessage = "failed to batch create order goods record"
		return primitive.CommitMessageState
	}
	saveOrderGoodsSpan.Finish()

	// delete shopping cart
	deleteShopCartSpan := opentracing.GlobalTracer().StartSpan("delete_shopcart", opentracing.ChildOf(parentSpan.Context()))

	if res := tx.Where(&model.ShoppingCart{User: orderInfo.User, Checked: true}).Delete(&model.ShoppingCart{}); res.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.ErrorMessage = "failed to delete ordered items in table of 'shoppingCart'"
		return primitive.CommitMessageState
	}
	deleteShopCartSpan.Finish()

	//发送延时消息
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{"127.0.0.1:9876"}),
		producer.WithInstanceName("TIMEOUT_PRODUCER"),
	)
	if err != nil {
		panic("生成producer失败")
	}

	//不要在一个进程中使用多个producer， 但是不要随便调用shutdown因为会影响其他的producer
	if err = p.Start(); err != nil {
		panic("启动producer失败")
	}

	msg = primitive.NewMessage("order_timeout", msg.Body)
	//msg.WithDelayTimeLevel(16)
	msg.WithDelayTimeLevel(5)
	_, err = p.SendSync(context.Background(), msg)
	if err != nil {
		zap.S().Errorf("发送延时消息失败: %v\n", err)
		tx.Rollback()
		o.Code = codes.Internal
		o.ErrorMessage = "发送延时消息失败"
		return primitive.CommitMessageState
	}

	tx.Commit()
	// If there are no errors in the process of creating a new order,
	// the message should be rolled back to prevent the stock quantity from being restored
	o.Code = codes.OK
	return primitive.RollbackMessageState
}

func (o *OrderListener) CheckLocalTransaction(ext *primitive.MessageExt) primitive.LocalTransactionState {
	var orderInfo model.OrderInfo
	_ = json.Unmarshal(ext.Body, &orderInfo)

	if res := global.DB.Where(&model.OrderInfo{OrderSn: orderInfo.OrderSn}).First(&orderInfo); res.RowsAffected == 0 {
		return primitive.CommitMessageState
	}
	return primitive.RollbackMessageState
}

func OrderTimeout(c context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		var orderInfo model.OrderInfo
		_ = json.Unmarshal(msgs[i].Body, &orderInfo)

		fmt.Printf("got order timeout msg: %v\n", time.Now())
		var order model.OrderInfo
		if result := global.DB.Model(model.OrderInfo{}).Where(model.OrderInfo{OrderSn: orderInfo.OrderSn}).First(&order); result.RowsAffected == 0 {
			return consumer.ConsumeSuccess, nil
		}

		if order.Status != "TRADE_SUCCESS" {
			tx := global.DB.Begin()
			//归还库存，我们可以模仿order中发送一个消息到 order_reback中去
			//修改订单的状态为已支付
			order.Status = "TRADE_CLOSED"
			tx.Save(&order)

			p, err := rocketmq.NewProducer(
				producer.WithNameServer([]string{"127.0.0.1:9876"}),
				producer.WithInstanceName("TIMEOUT_SELL_PRODUCER"),
			)
			if err != nil {
				panic("生成producer失败")
			}

			if err = p.Start(); err != nil {
				panic("启动producer失败")
			}

			_, err = p.SendSync(context.Background(), primitive.NewMessage("order_reback", msgs[i].Body))
			if err != nil {
				tx.Rollback()
				fmt.Printf("发送失败: %s\n", err)
				return consumer.ConsumeRetryLater, nil
			}

			//if err = p.Shutdown(); err != nil {panic("关闭producer失败")}
			return consumer.ConsumeSuccess, nil
		}

	}
	return consumer.ConsumeSuccess, nil
}
