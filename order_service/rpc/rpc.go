package rpc

import (
	"errors"
	"fmt"
	"order_service/config"
	"order_service/proto"

	_ "github.com/mbobakov/grpc-consul-resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 初始化其他服务的RPC客户端

var (
	GoodsCli proto.GoodsClient // 商品服务
	StoreCli proto.StoreClient // 库存服务
)

func InitSrvClient() error {
	if len(config.Conf.GoodsService.Name) == 0 {
		return errors.New("invalid GoodsService.Name")
	}
	if len(config.Conf.StoreService.Name) == 0 {
		return errors.New("invalid StoreService.Name")
	}
	// consul实现服务发现
	// 程序启动的时候请求consul获取一个可以用的商品服务地址
	goodsConn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=14s", config.Conf.ConsulConfig.Address, config.Conf.GoodsService.Name),
		// 指定round_robin策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("dial goods_srv failed, err:%v\n", err)
		return err
	}
	GoodsCli = proto.NewGoodsClient(goodsConn)

	stockConn, err := grpc.Dial(
		// consul服务
		fmt.Sprintf("consul://%s/%s?wait=14s", config.Conf.ConsulConfig.Address, config.Conf.StoreService.Name),
		// 指定round_robin策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("dial stock_srv failed, err:%v\n", err)
		return err
	}
	StoreCli = proto.NewStoreClient(stockConn)
	return nil
}
