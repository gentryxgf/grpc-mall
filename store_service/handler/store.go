package handler

import (
	"context"
	"encoding/json"
	"store_service/biz/store"
	"store_service/dao/mysql"
	"store_service/model"

	"store_service/proto"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StoreSrv struct {
	proto.UnimplementedStoreServer
}

func (s *StoreSrv) SetStore(ctx context.Context, req *proto.GoodsStoreInfo) (*proto.BaseResp, error) {
	if req.GetGoodsId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "GoodsId参数错误")
	}
	if req.GetNum() < 0 {
		return nil, status.Error(codes.InvalidArgument, "Num参数错误，不能为复数")
	}
	data, err := store.SetStoreByGoodsId(ctx, req.GetGoodsId(), req.GetNum())

	if err.Error() == "record not found" {
		zap.L().Error("GetStore failed:", zap.Int64("goods_id", req.GoodsId), zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, "未查询到商品Id")
	}
	if err != nil {
		zap.L().Error("SetStore failed:", zap.Int64("goods_id", req.GoodsId), zap.Int64("num", req.Num), zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}
	return data, nil
}

func (s *StoreSrv) GetStore(ctx context.Context, req *proto.GoodsStoreInfo) (*proto.GoodsStoreInfo, error) {
	if req.GetGoodsId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "参数错误")
	}
	data, err := store.GetStoreByGoodsId(ctx, req.GetGoodsId())
	if err != nil {
		zap.L().Error("GetStore failed:", zap.Int64("goods_id", req.GoodsId), zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}
	return data, nil
}

func (s *StoreSrv) ReduceStore(ctx context.Context, req *proto.GoodsStoreInfo) (*proto.GoodsStoreInfo, error) {
	if req.GetGoodsId() <= 0 || req.GetNum() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "无效的参数")
	}
	data, err := store.ReduceStore(ctx, req.GetGoodsId(), req.GetNum(), req.OrderId)
	if err != nil {
		zap.L().Error("ReduceStore failed:", zap.Int64("goods_id", req.GoodsId), zap.Error(err))
		return nil, status.Error(codes.Internal, "扣减库存失败")
	}
	return data, nil
}

func RollbackMsghandle(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		// zap.L().Info("hanlder.RollbackMsghandle start...")
		var data model.OrderGoodsStockInfo
		err := json.Unmarshal(msgs[i].Body, &data)
		if err != nil {
			zap.L().Error("json.Unmarshal RollbackMsg failed", zap.Error(err))
			continue
		}
		// 将库存回滚
		// zap.L().Info("mysql.RollbackStockByMsg start...")
		err = mysql.RollbackStockByMsg(ctx, data)
		// zap.L().Info("mysql.RollbackStockByMsg exit...")
		if err != nil {
			return consumer.ConsumeRetryLater, nil
		}
		return consumer.ConsumeSuccess, nil
	}
	return consumer.ConsumeSuccess, nil
}
