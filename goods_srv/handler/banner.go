package handler

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"mxshop_srvs/goods_srv/global"
	"mxshop_srvs/goods_srv/model"

	"google.golang.org/protobuf/types/known/emptypb"
	"mxshop_srvs/goods_srv/proto"
)

func (GoodsServer) BannerList(c context.Context, empty *emptypb.Empty) (*proto.BannerListResponse, error) {
	bannerListResponse := proto.BannerListResponse{}

	var banners []model.Banner
	result := global.DB.Find(&banners)
	bannerListResponse.Total = int32(result.RowsAffected)

	var bannerResponses []*proto.BannerResponse
	for _, banner := range banners {
		bannerResponses = append(bannerResponses, &proto.BannerResponse{
			Id:    banner.ID,
			Image: banner.Image,
			Index: banner.Index,
			Url:   banner.Url,
		})
	}

	bannerListResponse.Data = bannerResponses

	return &bannerListResponse, nil
}

func (GoodsServer) CreateBanner(c context.Context, r *proto.BannerRequest) (*proto.BannerResponse, error) {
	banner := model.Banner{}

	banner.Image = r.Image
	banner.Index = r.Index
	banner.Url = r.Url

	global.DB.Save(&banner)

	return &proto.BannerResponse{
		Id:    banner.ID,
		Image: banner.Image,
		Index: banner.Index,
		Url:   banner.Url,
	}, nil
}

func (GoodsServer) DeleteBanner(c context.Context, r *proto.BannerRequest) (*emptypb.Empty, error) {
	if result := global.DB.Delete(&model.Banner{}, r.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "轮播图不存在")
	}
	return &emptypb.Empty{}, nil
}

func (GoodsServer) UpdateBanner(c context.Context, r *proto.BannerRequest) (*emptypb.Empty, error) {
	var banner model.Banner

	if result := global.DB.First(&banner, r.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "轮播图不存在")
	}

	if r.Url != "" {
		banner.Url = r.Url
	}
	if r.Image != "" {
		banner.Image = r.Image
	}
	if r.Index != 0 {
		banner.Index = r.Index
	}

	global.DB.Save(&banner)

	return &emptypb.Empty{}, nil
}
