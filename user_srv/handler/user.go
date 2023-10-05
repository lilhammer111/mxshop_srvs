package handler

import (
	"context"
	"crypto/sha512"
	"fmt"
	"github.com/anaskhan96/go-password-encoder"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"mxshop_srvs/user_srv/global"
	"mxshop_srvs/user_srv/model"
	"mxshop_srvs/user_srv/proto"
	"strings"
	"time"
)

var Opts = &password.Options{SaltLen: 16, Iterations: 100, KeyLen: 32, HashFunction: sha512.New}

type UserServer struct {
	proto.UnimplementedUserServer
}

func Model2Response(user model.User) *proto.UserInfoResponse {
	userInfoRsp := proto.UserInfoResponse{
		Id:       user.ID,
		Password: user.Password,
		Mobile:   user.Mobile,
		NickName: user.NickName,
		Gender:   user.Gender,
		Role:     int32(user.Role),
	}
	if user.Birthday != nil {
		userInfoRsp.Birthday = uint64(user.Birthday.Unix())
	}
	return &userInfoRsp
}

func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}

		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func (*UserServer) GetUserList(ctx context.Context, req *proto.PageInfo) (*proto.UserListResponse, error) {
	var users []model.User
	res := global.DB.Find(&users)
	if res.Error != nil {
		return nil, res.Error
	}

	fmt.Println("用户列表")

	rsp := &proto.UserListResponse{}
	rsp.Total = int32(res.RowsAffected)

	global.DB.Scopes(Paginate(int(req.Pn), int(req.PSize))).Find(&users)

	for _, user := range users {
		userInfoRsp := Model2Response(user)
		rsp.Data = append(rsp.Data, userInfoRsp)
	}
	return rsp, nil
}

func (*UserServer) GetUserByMobile(ctx context.Context, req *proto.MobileRequest) (*proto.UserInfoResponse, error) {
	var user model.User
	res := global.DB.Where(&model.User{Mobile: req.Mobile}).First(&user)
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	userInfoRsp := Model2Response(user)
	return userInfoRsp, nil
}

func (*UserServer) GetUserById(ctx context.Context, req *proto.IdRequest) (*proto.UserInfoResponse, error) {
	var user model.User
	res := global.DB.First(&user, req.Id)
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}
	if res.Error != nil {
		return nil, res.Error
	}
	userInfoRsp := Model2Response(user)
	return userInfoRsp, nil
}

func (*UserServer) CreateUser(ctx context.Context, req *proto.CreateUserInfo) (*proto.UserInfoResponse, error) {
	// if user is existed
	var user model.User
	res := global.DB.Where(&model.User{Mobile: req.Mobile}).First(&user)
	if res.RowsAffected == 1 {
		return nil, status.Error(codes.AlreadyExists, "用户已存在")
	}

	// if not, create the user
	user.Mobile = req.Mobile
	user.NickName = req.NickName

	// encoded
	salt, encodedPwd := password.Encode(req.Password, Opts)
	user.Password = fmt.Sprintf("pbkdf2-sha512$%s$%s", salt, encodedPwd)

	// create user
	res = global.DB.Create(&user)
	if res.Error != nil {
		return nil, status.Error(codes.Internal, res.Error.Error())
	}

	userInfoRsp := Model2Response(user)
	return userInfoRsp, nil
}

func (*UserServer) UpdateUser(ctx context.Context, req *proto.UpdateUserInfo) (*emptypb.Empty, error) {
	// if user is not existed
	var user model.User
	res := global.DB.First(&user, req.Id)
	if res.RowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}

	birthday := time.Unix(int64(req.Birthday), 0)
	user.Birthday = &birthday
	user.NickName = req.NickName
	user.Gender = req.Gender
	res = global.DB.Save(user)
	if res.Error != nil {
		return nil, status.Error(codes.Internal, res.Error.Error())
	}

	return &emptypb.Empty{}, nil
}

func (*UserServer) CheckPassword(ctx context.Context, req *proto.PasswordCheckInfo) (*proto.CheckResponse, error) {
	pwdArr := strings.Split(req.EncryptedPassword, "$")
	check := password.Verify(req.Password, pwdArr[1], pwdArr[2], Opts)
	return &proto.CheckResponse{Success: check}, nil
}
