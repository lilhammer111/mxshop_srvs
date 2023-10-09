package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	res := global.DB.Where(&model.Category{Level: 1}).Preload("SubCategory.SubCategory").Find(&categories)

	//for _, category := range categories {
	//	fmt.Println(category.Name)
	//}
	b, _ := json.Marshal(&categories)

	return &proto.CategoryListResponse{JsonData: string(b), Total: int32(res.RowsAffected)}, nil

}

func (GoodsServer) GetSubCategory(c context.Context, r *proto.CategoryListRequest) (*proto.SubCategoryListResponse, error) {
	categoryListResponse := proto.SubCategoryListResponse{}
	var category model.Category
	if res := global.DB.First(&category, r.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "commerce is not existed")
	}

	categoryListResponse.Info = &proto.CategoryInfoResponse{
		Id:             category.ID,
		Name:           category.Name,
		ParentCategory: category.ParentCategoryID,
		Level:          category.Level,
		IsTab:          category.IsTab,
	}

	var subCategories []model.Category
	var subCategoriesResponses []*proto.CategoryInfoResponse
	preload := "SubCategory"

	if category.Level == 1 {
		preload = "SubCategory.SubCategory"
	}

	global.DB.Where(&model.Category{ParentCategoryID: r.Id}).Preload(preload).Find(&subCategories)
	for _, subCategory := range subCategories {
		subCategoriesResponses = append(subCategoriesResponses, &proto.CategoryInfoResponse{
			Id:             subCategory.ID,
			Name:           subCategory.Name,
			ParentCategory: subCategory.ParentCategoryID,
			Level:          subCategory.Level,
			IsTab:          subCategory.IsTab,
		})
	}

	categoryListResponse.SubCategorys = subCategoriesResponses
	return &categoryListResponse, nil
}

func (GoodsServer) CreateCategory(c context.Context, r *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	category := model.Category{}
	cMap := map[string]interface{}{}
	cMap["name"] = r.Name
	cMap["level"] = r.Level
	cMap["is_tab"] = r.IsTab
	if r.Level != 1 {
		//去查询父类目是否存在
		cMap["parent_category_id"] = r.ParentCategory
	}
	tx := global.DB.Model(&model.Category{}).Create(cMap)
	fmt.Println(tx)
	return &proto.CategoryInfoResponse{Id: int32(category.ID)}, nil
}

func (GoodsServer) DeleteCategory(c context.Context, r *proto.DeleteCategoryRequest) (*emptypb.Empty, error) {
	if result := global.DB.Delete(&model.Category{}, r.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	}
	return &emptypb.Empty{}, nil
}

func (GoodsServer) UpdateCategory(c context.Context, r *proto.CategoryInfoRequest) (*emptypb.Empty, error) {
	var category model.Category

	if result := global.DB.First(&category, r.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	}

	if r.Name != "" {
		category.Name = r.Name
	}
	if r.ParentCategory != 0 {
		category.ParentCategoryID = r.ParentCategory
	}
	if r.Level != 0 {
		category.Level = r.Level
	}
	if r.IsTab {
		category.IsTab = r.IsTab
	}

	global.DB.Save(&category)

	return &emptypb.Empty{}, nil
}
