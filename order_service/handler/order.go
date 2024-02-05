package handler

import (
	"context"
	"encoding/json"
	"order_service/biz/order"
	"order_service/config"
	"order_service/dao/mq"
	"order_service/dao/mysql"
	"order_service/model"
	"order_service/proto"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrderSrv struct {
	proto.UnimplementedOrderServer
}

func (s *OrderSrv) CreateOrder(ctx context.Context, req *proto.OrderReq) (*proto.OrderBaseResp, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "请求参数有误")
	}

	data, err := order.Create(ctx, req)
	if err != nil {
		zap.L().Error("order.Create failed:", zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return data, nil
}

// 延时消息的处理
func OrderTimeoutHandle(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		var data model.OrderGoodsStockInfo
		err := json.Unmarshal(msgs[i].Body, &data)
		if err != nil {
			zap.L().Error("OrderTimeoutHandle.json.Unmarshal failed", zap.Error(err))
			continue
		}
		o, err := mysql.QueryOrder(ctx, data.OrderId)
		if err != nil {
			zap.L().Error("mysql.QueryOrder failed", zap.Error(err))
			return consumer.ConsumeRetryLater, nil // 稍后再试
		}
		if o.OrderId == data.OrderId && o.Status == 100 { // 待支付
			msg := &primitive.Message{
				Topic: config.Conf.RocketMqConfig.Topic.StoreRollback,
				Body:  msgs[i].Body,
			}
			_, err = mq.Producer.SendSync(context.Background(), msg)
			if err != nil {
				zap.L().Error("send rollback msg failed", zap.Error(err))
				return consumer.ConsumeRetryLater, nil // 稍后再试
			}
			// 发送回滚库存成功，将订单状态设置为关闭
			o.Status = 300
			mysql.UpdateOrder(ctx, o)
		}
	}
	return consumer.ConsumeSuccess, nil
}
