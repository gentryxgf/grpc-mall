package main

import (
	"context"
	"fmt"
	"store_service/proto"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	conn   *grpc.ClientConn
	client proto.StoreClient
)

func init() {
	var err error
	conn, err = grpc.Dial("127.0.0.1:8382", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	client = proto.NewStoreClient(conn)
}

func TestReduceStore(wg *sync.WaitGroup) {
	defer wg.Done()
	param := &proto.GoodsStoreInfo{
		GoodsId: 1,
		Num:     1,
	}
	resp, err := client.ReduceStore(context.Background(), param)
	fmt.Printf("req:%v err:%v\n", resp, err)

}

func main() {
	defer conn.Close()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go TestReduceStore(&wg)
	}
	wg.Wait()

}
