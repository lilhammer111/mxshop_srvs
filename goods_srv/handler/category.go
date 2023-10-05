package handler

import (
	"context"
	"encoding/json"
	"google.golang.org/protobuf/types/known/emptypb"
	"mxshop_srvs/goods_srv/global"
	"mxshop_srvs/goods_srv/model"
	"mxshop_srvs/goods_srv/proto"
)

func (GoodsServer) GetAllCategorysList(c context.Context, empty *emptypb.Empty) (*proto.CategoryListResponse, error) {
	/*
		[
			{
				"id":xxx,
				"name":"",
				"level":1,
				"is_tab":false,
				"parent":13xxx,
				"sub_category":[
					"id":xxx,
					"name":"",
					"level":1,
					"is_tab":false,
					"sub_category":[]
				]
			}
		]
	*/
	var categories []model.Category

	global.DB.Where(&model.Category{Level: 1}).Preload("SubCategory.SubCategory").Find(&categories)

	//for _, category := range categories {
	//	fmt.Println(category.Name)
	//}

	b, _ := json.Marshal(categories)

	return &proto.CategoryListResponse{JsonData: string(b)}, nil

}

func (GoodsServer) GetSubCategory(c context.Context, r *proto.CategoryListRequest) (*proto.SubCategoryListResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (GoodsServer) CreateCategory(c context.Context, r *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (GoodsServer) DeleteCategory(c context.Context, r *proto.DeleteCategoryRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (GoodsServer) UpdateCategory(c context.Context, r *proto.CategoryInfoRequest) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}
