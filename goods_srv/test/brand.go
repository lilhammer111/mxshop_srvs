package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"mxshop_srvs/goods_srv/proto"
)

func TestGetBrandList() {
	rsp, err := brandCli.BrandList(context.Background(), &proto.BrandFilterRequest{})
	if err != nil {
		zap.S().Error("调用 BrandList 错误： ", err)
	}

	fmt.Println(rsp.Total)

	for _, brand := range rsp.Data {
		fmt.Println(brand.Name)
	}
}
