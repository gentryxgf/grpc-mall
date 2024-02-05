package order

import (
	"context"
	"encoding/json"
	"fmt"
	"order_service/config"
	"order_service/dao/mq"
	"order_service/dao/mysql"
	"order_service/model"
	"order_service/proto"
	"order_service/rpc"
	"order_service/third_party/snowflake"
	"strconv"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type OrderMessageEntity struct {
	OrderId int64
	Param   *proto.OrderReq
	err     error
}

func (o *OrderMessageEntity) ExecuteLocalTransaction(*primitive.Message) primitive.LocalTransactionState {
	// 事务消息整体执行流程：
	// 1.生产者先向RocketMQ发送Half消息
	// 2.向RocketMQ发送的Half消息得到回复后开始执行本地事务
	// 3.本地事务执行成功向RocketMQ回复rollback， 执行失败向RocketMQ回复commit
	// 4.RocketMQ根据步骤3的结果，如果为rollback，则丢弃消息，如果为commit，则真正的进行消息投递
	// 5.消费端消费消息

	// 在这个例子中，我们的Half消息为扣减库存的消息
	// 如果本地创建订单的事务执行成功，表示库存也扣减了，所有都执行正确，就只需回复RocketMQ rollback，消息会被丢弃，不会传给下游消费端
	// 如果本地订单创建失败，但是库存已经扣减了，这时就需要回滚库存，就需要向RocketMQ回复commit，此时消息才会被真正给下游的消费端消费

	// 如果本地事务执行成功，则不向RocketMQ发送扣减库存的消息
	fmt.Println("Start ExecuteLocalTransaction...")
	// 检查参数
	// 参数错误，此时库存还未扣减，则丢弃回滚库存的消息，所以回复rollback，消息被丢弃
	if o.Param == nil {
		zap.L().Error("ExecuteLocalTransation param is nil")
		o.err = status.Error(codes.Internal, "invalid OrderMessageEntity")
		return primitive.RollbackMessageState
	}
	params := o.Param
	ctx := context.Background()
	// 查询商品详情，此时也还没扣减库存，如果出错，则丢弃回滚库存的消息，所以回复rollback，消息被丢弃
	goodsDetail, err := rpc.GoodsCli.GetGoodsDetail(ctx, &proto.GetGoodsDetailReq{
		GoodsId: params.GoodsId,
		UserId:  params.UserId,
	})
	if err != nil {
		zap.L().Error("rpc.GoodsCli.GetGoodsDetail failed", zap.Error(err))
		o.err = status.Error(codes.Internal, err.Error())
		return primitive.RollbackMessageState
	}

	payAmountStr := goodsDetail.Price
	payAmount, _ := strconv.ParseFloat(payAmountStr, 64)
	payAmount *= float64(params.Num)
	// 扣减库存，此时库存也还没完成扣减，如果出错，同样丢弃回滚库存的消息，所以回复rollback，消息丢弃
	_, err = rpc.StoreCli.ReduceStore(ctx, &proto.GoodsStoreInfo{
		GoodsId: params.GoodsId,
		Num:     params.Num,
		OrderId: o.OrderId,
	})
	if err != nil {
		zap.L().Error("rpc.StoreCli.ReduceStore failed", zap.Error(err))
		o.err = status.Error(codes.Internal, err.Error())
		return primitive.RollbackMessageState
	}

	orderData := model.Order{
		OrderId:        o.OrderId,
		UserId:         params.UserId,
		PayAmount:      int64(payAmount),
		ReceiveAddress: params.Address,
		ReceiveName:    params.Name,
		ReceivePhone:   params.Phone,
		Status:         100,
	}

	orderDetail := model.OrderDetail{
		OrderId: o.OrderId,
		UserId:  params.UserId,
		GoodsId: params.GoodsId,
		Num:     params.Num,
	}
	// 创建订单
	// 此时库存已经扣减，如果再出错，就需要回滚库存了，需要向RocketMQ回复commit，使消息被真正投递出去
	err = mysql.CreateOrderWithTransation(ctx, &orderData, &orderDetail)
	// err = errors.New("my error")
	if err != nil {
		zap.L().Error("mysql.CreateOrderWithTransation fail", zap.Error(err))
		o.err = status.Error(codes.Internal, err.Error())
		return primitive.CommitMessageState
	}

	// 发送延时消息
	// 向延时的topic发送消息，经过指定的时间才能会被下游的消费端进行消费
	// 如果订单超过指定时间，就取消订单，需要向延时的topic投递回滚库存的消息
	data := model.OrderGoodsStockInfo{
		OrderId: o.OrderId,
		GoodsId: params.GoodsId,
		Num:     params.Num,
	}

	b, _ := json.Marshal(data)
	msg := primitive.NewMessage(config.Conf.RocketMqConfig.Topic.PayTimeout, b)
	// 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
	msg.WithDelayTimeLevel(2)
	_, err = mq.Producer.SendSync(context.Background(), msg)
	// 这里还是本地事务的执行流程，如果出错了，需要回复commit，因为前面已经扣减过了，需要回滚库存
	if err != nil {
		zap.L().Error("mq.Producer.SendSync failed", zap.Error(err))
		return primitive.CommitMessageState
	}
	// 执行到这里，表明所有的步骤都已经完成，本地事务执行成功，所以不需要回滚库存，回复rollback将回滚库存的消息进行丢弃
	return primitive.RollbackMessageState

}

func (o *OrderMessageEntity) CheckLocalTransaction(*primitive.MessageExt) primitive.LocalTransactionState {
	// 本地事务回查
	// 当RocketMQ没有收到生产者执行本地事务的状态的时候，执行本地事务的回查
	// 当订单创建成功时，说明本地事务执行成功了，就不需要回滚库存
	// 如果订单创建失败，则说明本地事务没有执行成功，就需要回滚库存
	_, err := mysql.QueryOrder(context.Background(), o.OrderId)
	if err == gorm.ErrRecordNotFound {
		return primitive.CommitMessageState
	}
	return primitive.RollbackMessageState
}

func Create(ctx context.Context, params *proto.OrderReq) (*proto.OrderBaseResp, error) {
	orderId := snowflake.GenID()
	orderEntity := &OrderMessageEntity{
		OrderId: orderId,
		Param:   params,
	}

	p, err := rocketmq.NewTransactionProducer(
		orderEntity,
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"})),
		producer.WithRetry(2),
		producer.WithGroupName("order_srv_1"),
	)

	if err != nil {
		zap.L().Error("rocketmq.NewTransactionProducer failed", zap.Error(err))
		return nil, err
	}

	p.Start()
	defer p.Shutdown()

	// 封装消息 orderId GoodsId num
	data := model.OrderGoodsStockInfo{
		OrderId: orderId,
		GoodsId: params.GoodsId,
		Num:     params.Num,
	}
	b, _ := json.Marshal(data)
	msg := &primitive.Message{
		Topic: config.Conf.RocketMqConfig.Topic.StoreRollback, // xx_stock_rollback
		Body:  b,
	}
	// 发送事务消息
	res, err := p.SendMessageInTransaction(context.Background(), msg)
	if err != nil {
		zap.L().Error("p.SendMessageInTransaction failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create order failed")
	}
	zap.L().Info("p.SendMessageInTransaction success", zap.Any("res", res))
	// 如果回滚库存的消息被投递出去（commit）说明本地事务执行失败，也就是创建订单失败
	if res.State == primitive.CommitMessageState {
		return nil, status.Error(codes.Internal, "create order failed")
	}
	// 其他内部错误
	if orderEntity.err != nil {
		return nil, orderEntity.err
	}
	resp := &proto.OrderBaseResp{
		Code: int32(codes.OK),
		Msg:  "创建订单成功",
	}
	return resp, nil

}

/* func Create(ctx context.Context, params *proto.OrderReq) (*proto.OrderBaseResp, error) {
	orderId := snowflake.GenID()
	goodsDetail, err := rpc.GoodsCli.GetGoodsDetail(ctx, &proto.GetGoodsDetailReq{
		GoodsId: params.GoodsId,
		UserId:  params.UserId,
	})
	if err != nil {
		zap.L().Error("rpc.GoodsCli.GetGoodsDetail failed", zap.Error(err))
		return nil, err
	}

	payAmountStr := goodsDetail.Price
	payAmount, _ := strconv.ParseFloat(payAmountStr, 64)
	payAmount *= float64(params.Num)

	_, err = rpc.StoreCli.ReduceStore(ctx, &proto.GoodsStoreInfo{
		GoodsId: params.GoodsId,
		Num:     params.Num,
	})
	if err != nil {
		zap.L().Error("rpc.StoreCli.ReduceStore failed", zap.Error(err))
		return nil, err
	}

	orderData := model.Order{
		OrderId:        orderId,
		UserId:         params.UserId,
		PayAmount:      int64(payAmount),
		ReceiveAddress: params.Address,
		ReceiveName:    params.Name,
		ReceivePhone:   params.Phone,
	}

	orderDetail := model.OrderDetail{
		OrderId: orderId,
		UserId:  params.UserId,
		GoodsId: params.GoodsId,
		Num:     params.Num,
	}

	err = mysql.CreateOrderWithTransation(ctx, &orderData, &orderDetail)
	if err != nil {
		zap.L().Error("mysql.CreateOrderWithTransation fail", zap.Error(err))
		return nil, err
	}

	resp := &proto.OrderBaseResp{
		Code: int32(codes.OK),
		Msg:  "创建订单成功",
	}
	return resp, nil
} */
