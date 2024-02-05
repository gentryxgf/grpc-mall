package handler

import (
	"context"
	"good_service/biz/goods"
	"good_service/proto"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GoodSrv struct {
	proto.UnimplementedGoodsServer
}

// GetGoodsByRoom 根据room_id获取直播间商品列表
func (s *GoodSrv) GetGoodsByRoom(ctx context.Context, req *proto.GetGoodsByRoomReq) (*proto.GoodsListResp, error) {
	// 参数处理
	if req.GetRoomId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "请求参数有误")
	}

	// 查询数据并封装返回的响应数据
	data, err := goods.GetGoodsByRoomId(ctx, req.GetRoomId())
	if err != nil {
		zap.L().Error("goods.GetGoodsByRoomId failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return data, nil
}

func (s *GoodSrv) GetGoodsDetail(ctx context.Context, req *proto.GetGoodsDetailReq) (*proto.GoodsDetail, error) {
	if req.GetGoodsId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "请求参数有误")
	}

	data, err := goods.GetGoodsDetail(ctx, req.GetGoodsId())
	if err != nil {
		zap.L().Error("goods.GetGoodDetail failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}
	return data, nil
}
