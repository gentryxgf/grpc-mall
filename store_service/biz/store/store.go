package store

import (
	"context"
	"store_service/dao/mysql"
	"store_service/proto"

	"google.golang.org/grpc/codes"
)

func GetStoreByGoodsId(ctx context.Context, goodsId int64) (*proto.GoodsStoreInfo, error) {
	data, err := mysql.GetStoreByGoodsId(ctx, goodsId)
	if err != nil {
		return nil, err
	}

	resp := &proto.GoodsStoreInfo{
		GoodsId: data.GoodsId,
		Num:     data.Num,
	}

	return resp, nil
}

func SetStoreByGoodsId(ctx context.Context, goodsId, num int64) (*proto.BaseResp, error) {
	err := mysql.SetStoreByGoodsId(ctx, goodsId, num)
	if err != nil {
		return nil, err
	}
	resp := &proto.BaseResp{
		Code: int32(codes.OK),
		Msg:  "设置成功",
	}
	return resp, nil
}

func ReduceStore(ctx context.Context, goodsId, num, orderId int64) (*proto.GoodsStoreInfo, error) {
	data, err := mysql.ReduceStore(ctx, goodsId, num, orderId)
	if err != nil {
		return nil, err
	}

	resp := &proto.GoodsStoreInfo{
		GoodsId: data.GoodsId,
		Num:     data.Num,
		OrderId: orderId,
	}

	return resp, nil
}
