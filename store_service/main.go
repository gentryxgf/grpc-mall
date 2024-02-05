package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"store_service/config"
	"store_service/dao/mysql"
	"store_service/dao/redis"
	"store_service/handler"
	"store_service/logger"
	"store_service/proto"
	"store_service/registry"
	"syscall"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	var cfn string
	flag.StringVar(&cfn, "conf", "./conf/config.yaml", "指定配置文件")
	flag.Parse()

	if err := config.Init(cfn); err != nil {
		panic(err)
	}

	if err := logger.Init(config.Conf.LogConfig, config.Conf.Mode); err != nil {
		panic(err)
	}

	if err := mysql.Init(config.Conf.MySQLConfig); err != nil {
		panic(err)
	}

	if err := redis.Init(config.Conf.RedisConfig); err != nil {
		panic(err)
	}

	if err := registry.Init(config.Conf.ConsulConfig.Address); err != nil {
		panic(err)
	}

	// 监听库存回滚的消息
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName("store_srv_1"),
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"})),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
		consumer.WithConsumerOrder(true),
	)
	/* c, _ := rocketmq.NewPushConsumer(
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"})),
	) */
	// 订阅topic
	err := c.Subscribe("xx_store_rollback", consumer.MessageSelector{}, handler.RollbackMsghandle)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Note: start after subscribe
	err = c.Start()
	if err != nil {
		panic(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RpcPort))
	if err != nil {
		panic(err)
	}

	s := grpc.NewServer()
	// 注册健康检查服务，至此consul来对我进行检查
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	proto.RegisterStoreServer(s, &handler.StoreSrv{})

	go func() {
		err := s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
	registry.Reg.RegisterService(config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort, nil)
	zap.L().Sugar().Infof("Service start at consul: %s-%s-%d", config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort)

	zap.L().Sugar().Infof("RPC sever start at: %s:%d", config.Conf.Ip, config.Conf.RpcPort)

	conn, err := grpc.DialContext(
		context.Background(),
		fmt.Sprintf("%s:%d", config.Conf.Ip, config.Conf.RpcPort),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		zap.L().Info("Fail to dial server:", zap.Error(err))
	}

	gwmux := runtime.NewServeMux()

	err = proto.RegisterStoreHandler(context.Background(), gwmux, conn)
	if err != nil {
		zap.L().Info("Fail to register gateway:", zap.Error(err))
	}

	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Conf.HttpPort),
		Handler: gwmux,
	}

	zap.S().Infof("Serving gRPc-GateWay on http://0.0.0.0:%d", config.Conf.HttpPort)

	go func() {
		err := gwServer.ListenAndServe()
		if err != nil {
			zap.S().Infof("gwServer.ListenAndServe failed, err:%v", err)
			return
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	serviceId := fmt.Sprintf("%s-%s-%d", config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort)
	registry.Reg.Deregister(serviceId)
}
