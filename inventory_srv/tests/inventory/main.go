package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"mxshop_srvs/inventory_srv/proto"
	"sync"
)

var inventoryClient proto.InventoryClient
var conn *grpc.ClientConn

func Init() {
	var err error
	conn, err = grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	inventoryClient = proto.NewInventoryClient(conn)
}

func TestSetInv(goodsId, num int32) {
	_, err := inventoryClient.SetInv(context.Background(), &proto.GoodsInvInfo{
		GoodsId: goodsId,
		Num:     num,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("setting inventory succeed")

}

func TestInvDetail(goodsID int32) {
	rsp, err := inventoryClient.InvDetail(context.Background(), &proto.GoodsInvInfo{
		GoodsId: goodsID,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp.GoodsId, rsp.Num)
}

func TestSell() {
	_, err := inventoryClient.Sell(context.Background(), &proto.SellInfo{
		GoodsInfo: []*proto.GoodsInvInfo{
			{
				GoodsId: 421,
				Num:     10,
			},
			{
				GoodsId: 422,
				Num:     10,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("batch deduct successfully")
}

func TestSell1(wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := inventoryClient.Sell(context.Background(), &proto.SellInfo{
		GoodsInfo: []*proto.GoodsInvInfo{
			{
				GoodsId: 421,
				Num:     1,
			},
			{
				GoodsId: 422,
				Num:     1,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("batch deduct successfully")
}

func TestReback() {
	_, err := inventoryClient.Reback(context.Background(), &proto.SellInfo{
		GoodsInfo: []*proto.GoodsInvInfo{
			{
				GoodsId: 421,
				Num:     10,
			},
			{
				GoodsId: 422,
				Num:     10,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("return inventory successfully")
}

func main() {
	Init()
	defer conn.Close()
	//TestSetInv(422, 40)
	//TestInvDetail(422)
	//TestSell()
	//TestReback()

	//var i int32
	//for i = 421; i <= 840; i++ {
	//	TestSetInv(i, 100)
	//}
	//

	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go TestSell1(&wg)
	}
	wg.Wait()

}
