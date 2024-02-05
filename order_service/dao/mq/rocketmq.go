package mq

import (
	"order_service/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
)

var (
	Producer rocketmq.Producer
)

func Init(cfg *config.RocketMqConfig) (err error) {
	Producer, err = rocketmq.NewProducer(
		//producer.WithNameServer(endPoint),
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{cfg.Addr})),
		producer.WithRetry(2),
		producer.WithGroupName(cfg.GroupId),
	)
	if err != nil {
		zap.L().Error("rocketmq.NewProducer failed", zap.Error(err))
		return
	}

	err = Producer.Start()
	if err != nil {
		zap.L().Error("Producer.Start failed", zap.Error(err))
		return
	}
	return nil
}

func Exit() error {
	err := Producer.Shutdown()
	if err != nil {
		zap.L().Error("Producer.Shutdown failed", zap.Error(err))
		return err
	}
	return nil
}
