package handler

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"mxshop_srvs/goods_srv/global"
	"mxshop_srvs/goods_srv/model"
	"mxshop_srvs/goods_srv/proto"
)

func (GoodsServer) BrandList(c context.Context, request *proto.BrandFilterRequest) (*proto.BrandListResponse, error) {
	var brands []model.Brand
	var brandListRsp proto.BrandListResponse
	res := global.DB.Scopes(Paginate(int(request.Pages), int(request.PagePerNums))).Find(&brands)
	if res.Error != nil {
		return nil, res.Error
	}

	var total int64
	global.DB.Model(&model.Brand{}).Count(&total)
	brandListRsp.Total = int32(total)

	//fmt.Println(res.RowsAffected)
	var brandRsps []*proto.BrandInfoResponse
	for _, brand := range brands {
		brandRsps = append(brandRsps, &proto.BrandInfoResponse{
			Id:   brand.ID,
			Name: brand.Name,
			Logo: brand.Logo,
		})
	}
	brandListRsp.Data = brandRsps
	return &brandListRsp, nil
}

func (GoodsServer) CreateBrand(c context.Context, r *proto.BrandRequest) (*proto.BrandInfoResponse, error) {

	if res := global.DB.First(&model.Brand{}); res.RowsAffected == 1 {
		return nil, status.Error(codes.InvalidArgument, "品牌已存在")
	}

	brand := model.Brand{Logo: r.Logo, Name: r.Name}

	if res := global.DB.Save(&brand); res.Error != nil {
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return &proto.BrandInfoResponse{
		Id:   brand.ID,
		Name: brand.Name,
		Logo: brand.Logo,
	}, nil
}

func (GoodsServer) DeleteBrand(c context.Context, r *proto.BrandRequest) (*emptypb.Empty, error) {
	if res := global.DB.Delete(&model.Brand{}, r.Id); res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "品牌不存在")
	}

	return &emptypb.Empty{}, nil
}

func (GoodsServer) UpdateBrand(c context.Context, r *proto.BrandRequest) (*emptypb.Empty, error) {
	brand := model.Brand{}

	if res := global.DB.Delete(&brand, r.Id); res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "品牌不存在")
	}

	if r.Name != "" {
		brand.Name = r.Name
	}

	if r.Logo != "" {
		brand.Logo = r.Logo
	}

	if res := global.DB.Save(&brand); res.Error != nil {
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return &emptypb.Empty{}, nil

}
