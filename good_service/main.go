package main

import (
	"context"
	"flag"
	"fmt"
	"good_service/config"
	"good_service/dao/mysql"
	"good_service/handler"
	"good_service/logger"
	"good_service/proto"
	"good_service/registry"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	var cfn string
	//命令行解析配置文件
	// good_service -conf="./conf/config.yaml"
	flag.StringVar(&cfn, "conf", "./conf/config.yaml", "指定配置文件路径")
	flag.Parse()

	// 加载配置文件
	err := config.Init(cfn)
	if err != nil {
		// 配置文件加载失败直接退出
		panic(err)
	}

	// 加载日志
	err = logger.Init(config.Conf.LogConfig, config.Conf.Mode)
	if err != nil {
		// 日志加载失败直接退出
		panic(err)
	}

	// MySQL初始化
	err = mysql.Init(config.Conf.MySQLConfig)
	if err != nil {
		// MySQL初始化失败直接退出
		panic(err)
	}

	// consul 初始化
	err = registry.Init(config.Conf.ConsulConfig.Address)
	if err != nil {
		panic(err)
	}

	// 监听端口
	// 不写127.0.0.1，只写端口号，不然外部访问不到你这个rpc服务，同时consul也就无法执行健康检查
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RpcPort))
	if err != nil {
		panic(err)
	}

	// 创建gRPC服务
	s := grpc.NewServer()
	// 注册健康检查服务，至此consul来对我进行检查
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	// 商品服务注册RPC服务
	proto.RegisterGoodsServer(s, &handler.GoodSrv{})

	// 启动gRPC服务
	go func() {
		err := s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()
	// 注册服务到consul
	registry.Reg.RegisterService(config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort, nil)

	zap.L().Sugar().Infof("Service start at consul: %s-%s-%d", config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort)

	// 创建grpc客户端
	conn, err := grpc.DialContext(
		context.Background(),
		fmt.Sprintf("%s:%d", config.Conf.Ip, config.Conf.RpcPort),
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		zap.L().Fatal("Failed to dial server:", zap.Error(err))
	}

	gwmux := runtime.NewServeMux()
	err = proto.RegisterGoodsHandler(context.Background(), gwmux, conn)
	if err != nil {
		zap.L().Fatal("Failed to register gatewary:", zap.Error(err))
	}

	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Conf.HttpPort),
		Handler: gwmux,
	}
	zap.L().Sugar().Infof("Serving gRPC-GateWay on http: 0.0.0.0%s", gwServer.Addr)

	go func() {
		err := gwServer.ListenAndServe()
		if err != nil {
			zap.L().Info("gwServer.ListenAndServe failed, err:", zap.Error(err))
		}
		return
	}()

	// 服务退出时要注销服务
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	// 退出服务时注销服务
	serviceId := fmt.Sprintf("%s-%s-%d", config.Conf.Name, config.Conf.Ip, config.Conf.RpcPort)
	registry.Reg.Deregister(serviceId)
}
