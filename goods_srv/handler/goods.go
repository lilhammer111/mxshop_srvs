package handler

import (
	"context"
	"encoding/json"
	"github.com/olivere/elastic/v7"
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

	q := elastic.NewBoolQuery()

	if r.KeyWords != "" {
		q = q.Must(elastic.NewMultiMatchQuery(r.KeyWords, "name", "goods_brief "))

	}
	if r.IsHot {
		q = q.Filter(elastic.NewTermQuery("is_hot", r.IsHot))
	}
	if r.IsNew {
		q = q.Filter(elastic.NewTermQuery("is_new", r.IsNew))
	}
	if r.PriceMin > 0 {
		q = q.Filter(elastic.NewRangeQuery("shop_price").Gte(r.PriceMin))
	}
	if r.PriceMax > 0 {
		q = q.Filter(elastic.NewRangeQuery("shop_price").Lte(r.PriceMax))
	}
	if r.Brand > 0 {
		q = q.Filter(elastic.NewTermQuery("brand_id", r.Brand))
	}

	// query CategoryID
	var subQuery string
	categoryIds := make([]interface{}, 0)
	if r.TopCategory > 0 {
		var category model.Category
		if result := global.DB.First(&category, r.TopCategory); result.RowsAffected == 0 {
			return nil, status.Errorf(codes.NotFound, "商品分类不存在")
		}

		if category.Level == 1 {
			subQuery = "select id from category where parent_category_id in (select id from category WHERE parent_category_id=?)"
		} else if category.Level == 2 {
			subQuery = "select id from category WHERE parent_category_id=?"
		} else if category.Level == 3 {
			subQuery = "select id from category WHERE id=?"
		}

		type Result struct {
			ID int32
		}
		var results []Result
		global.DB.Model(model.Category{}).Raw(subQuery, r.TopCategory).Scan(&results)
		for _, re := range results {
			categoryIds = append(categoryIds, re.ID)
		}

		//生成terms查询
		q = q.Filter(elastic.NewTermsQuery("category_id", categoryIds...))
	}

	// paginate
	if r.Pages == 0 {
		r.Pages = 1
	}
	switch {
	case r.PagePerNums > 100:
		r.PagePerNums = 100
	case r.PagePerNums <= 0:
		r.PagePerNums = 10
	}
	// es query result
	result, err := global.EsClient.Search().Index(model.EsGoods{}.GetIndexName()).Query(q).From(int(r.Pages)).Size(int(r.PagePerNums)).Do(context.Background())
	if err != nil {
		return nil, err
	}
	// query goods primary key
	glr := &proto.GoodsListResponse{}

	glr.Total = int32(result.Hits.TotalHits.Value)

	goodsIds := make([]int32, 0)
	for _, value := range result.Hits.Hits {
		goods := model.EsGoods{}
		err = json.Unmarshal(value.Source, &goods)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal es result error")
		}

		goodsIds = append(goodsIds, goods.ID)
	}

	// query all goods in goodsIds by mysql
	var goods []model.Goods
	res := global.DB.Preload("Category").Preload("Brands").Find(&goods, goodsIds)
	if res.Error != nil {
		return nil, res.Error
	}

	// model to response
	for _, g := range goods {
		gir := ModelToResponse(g)
		glr.Data = append(glr.Data, &gir)
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

	goods := model.Goods{
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

	tx := global.DB.Begin()
	if res := tx.Save(&goods); res.Error != nil {
		tx.Rollback()
		return nil, status.Errorf(codes.InvalidArgument, "fail to add commodity")
	}
	tx.Commit()

	return &proto.GoodsInfoResponse{Id: goods.ID}, nil

}

func (GoodsServer) DeleteGoods(c context.Context, info *proto.DeleteGoodsInfo) (*emptypb.Empty, error) {
	// todo Does the deletion of a goods also need to be synchronized in es?
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

	tx := global.DB.Begin()
	result := tx.Save(&goods)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	tx.Commit()
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
