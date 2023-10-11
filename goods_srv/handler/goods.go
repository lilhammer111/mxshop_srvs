package handler

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"mxshop_srvs/goods_srv/global"
	"mxshop_srvs/goods_srv/model"
	"mxshop_srvs/goods_srv/proto"
)

type GoodsServer struct {
	proto.UnimplementedGoodsServer
}

func ModelToResponse(goods model.Goods) proto.GoodsInfoResponse {
	return proto.GoodsInfoResponse{
		Id:              goods.ID,
		CategoryId:      goods.CategoryID,
		Name:            goods.Name,
		GoodsSn:         goods.GoodsSn,
		ClickNum:        goods.ClickNum,
		SoldNum:         goods.SoldNum,
		FavNum:          goods.FavNum,
		MarketPrice:     goods.MarketPrice,
		ShopPrice:       goods.ShopPrice,
		GoodsBrief:      goods.GoodsBrief,
		ShipFree:        goods.ShipFree,
		GoodsFrontImage: goods.GoodsFrontImage,
		IsNew:           goods.IsNew,
		IsHot:           goods.IsHot,
		OnSale:          goods.OnSale,
		DescImages:      goods.DescImages,
		Images:          goods.Images,
		Category: &proto.CategoryBriefInfoResponse{
			Id:   goods.Category.ID,
			Name: goods.Category.Name,
		},
		Brand: &proto.BrandInfoResponse{
			Id:   goods.Brands.ID,
			Name: goods.Brands.Name,
			Logo: goods.Brands.Logo,
		},
	}
}

func (GoodsServer) GoodsList(c context.Context, r *proto.GoodsFilterRequest) (*proto.GoodsListResponse, error) {
	glr := &proto.GoodsListResponse{}

	var goods []model.Goods

	localDB := global.DB.Model(model.Goods{})
	if r.KeyWords != "" {
		localDB = localDB.Where("name like ?", "%"+r.KeyWords+"%")
	}
	if r.IsHot {
		localDB = localDB.Where(model.Goods{IsHot: true})
	}
	if r.IsNew {
		localDB = localDB.Where(model.Goods{IsNew: true})
	}
	if r.PriceMin > 0 {
		localDB = localDB.Where("shop_price >= ?", r.PriceMin)
	}
	if r.PriceMax > 0 {
		localDB = localDB.Where("shop_price <= ?", r.PriceMax)
	}
	if r.Brand > 0 {
		localDB = localDB.Where("brand_id = ?", r.Brand)
	}

	var subQuery string

	if r.TopCategory > 0 {
		var category model.Category
		if res := global.DB.First(&category, r.TopCategory); res.RowsAffected == 0 {
			return nil, status.Errorf(codes.NotFound, "commodity does not exist")
		}

		if category.Level == 1 {
			subQuery = fmt.Sprintf("select id from category where parent_category_id in (select id from category where parent_category_id=%d)", r.TopCategory)
		} else if category.Level == 2 {
			subQuery = fmt.Sprintf("select id from category where parent_category_id=%d", r.TopCategory)
		} else if category.Level == 3 {
			subQuery = fmt.Sprintf("select id from category where id =%d", r.TopCategory)
		}
		localDB = localDB.Where(fmt.Sprintf("category_id in (%s)", subQuery)).Find(&goods)
	}
	// calculate "total" value
	var total int64
	localDB.Count(&total)
	glr.Total = int32(total)

	res := localDB.Preload("Brands").Preload("Category").Scopes(Paginate(int(r.Pages), int(r.PagePerNums))).Find(&goods)
	if res.Error != nil {
		return nil, res.Error
	}

	for _, good := range goods {
		goodsInfoResponse := ModelToResponse(good)
		glr.Data = append(glr.Data, &goodsInfoResponse)
	}

	return glr, nil
}

func (GoodsServer) BatchGetGoods(c context.Context, info *proto.BatchGoodsIdInfo) (*proto.GoodsListResponse, error) {
	glr := &proto.GoodsListResponse{}
	var goods []model.Goods

	// gorm case:
	// Slice of primary key
	// db.Where([]int64{20, 21, 22}).Find(&users)
	// SELECT * FROM users WHERE id IN (20, 21, 22);
	res := global.DB.Where(info.Id).Find(&goods)

	for _, good := range goods {
		goodsInfoResponse := ModelToResponse(good)
		glr.Data = append(glr.Data, &goodsInfoResponse)
	}
	glr.Total = int32(res.RowsAffected)
	return glr, nil
}

func (GoodsServer) CreateGoods(c context.Context, info *proto.CreateGoodsInfo) (*proto.GoodsInfoResponse, error) {
	var category model.Category
	if result := global.DB.First(&category, info.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "commodity category does not exist")
	}

	var brand model.Brands
	if result := global.DB.First(&brand, info.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "brand does not exist")
	}

	good := model.Goods{
		Stocks:          info.Stocks,
		CategoryID:      category.ID,
		Category:        category,
		BrandsID:        brand.ID,
		Brands:          brand,
		OnSale:          info.OnSale,
		ShipFree:        info.ShipFree,
		IsNew:           info.IsNew,
		IsHot:           info.IsHot,
		Name:            info.Name,
		GoodsSn:         info.GoodsSn,
		MarketPrice:     info.MarketPrice,
		ShopPrice:       info.ShopPrice,
		GoodsBrief:      info.GoodsBrief,
		Images:          info.Images,
		DescImages:      info.DescImages,
		GoodsFrontImage: info.GoodsFrontImage,
	}
	// err
	//if res := global.DB.Preload("Category").Preload("Brands").Create(&good); res.RowsAffected == 0 {
	//	return nil, status.Errorf(codes.InvalidArgument, "fail to add commodity")
	//}

	if res := global.DB.Save(&good); res.Error != nil {
		return nil, status.Errorf(codes.InvalidArgument, "fail to add commodity")
	}
	goodsInfoResponse := ModelToResponse(good)

	return &goodsInfoResponse, nil

}

func (GoodsServer) DeleteGoods(c context.Context, info *proto.DeleteGoodsInfo) (*emptypb.Empty, error) {
	if res := global.DB.Delete(&model.Goods{}, info.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "commodity does not exist")
	}

	return &emptypb.Empty{}, nil
}

func (GoodsServer) UpdateGoods(c context.Context, info *proto.CreateGoodsInfo) (*emptypb.Empty, error) {
	var goods model.Goods

	if result := global.DB.First(&goods, info.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "commodity does not exist")
	}

	var category model.Category
	if result := global.DB.First(&category, goods.CategoryID); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "commodity category does not exist")
	}

	var brand model.Brands
	if result := global.DB.First(&brand, goods.Brands); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "brand does not exist")
	}

	goods.Brands = brand
	goods.BrandsID = brand.ID
	goods.Category = category
	goods.CategoryID = category.ID
	goods.Name = info.Name
	goods.GoodsSn = info.GoodsSn
	goods.MarketPrice = info.MarketPrice
	goods.ShopPrice = info.ShopPrice
	goods.GoodsBrief = info.GoodsBrief
	goods.ShipFree = info.ShipFree
	goods.Images = info.Images
	goods.DescImages = info.DescImages
	goods.GoodsFrontImage = info.GoodsFrontImage
	goods.IsNew = info.IsNew
	goods.IsHot = info.IsHot
	goods.OnSale = info.OnSale

	//tx := global.DB.Begin()
	//result := tx.Save(&goods)
	//if result.Error != nil {
	//	tx.Rollback()
	//	return nil, result.Error
	//}
	//tx.Commit()

	global.DB.Save(&goods)
	return &emptypb.Empty{}, nil
}

func (GoodsServer) GetGoodsDetail(c context.Context, r *proto.GoodInfoRequest) (*proto.GoodsInfoResponse, error) {
	var good model.Goods
	if res := global.DB.Preload("Category").Preload("Brands").First(&good, r.Id); res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "commodity does not exist")
	}
	goodsInfoResponse := ModelToResponse(good)
	return &goodsInfoResponse, nil
}
