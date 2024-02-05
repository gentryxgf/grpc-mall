package goods

import (
	"context"
	"encoding/json"
	"fmt"
	"good_service/dao/mysql"
	"good_service/proto"
)

func GetGoodsDetail(ctx context.Context, goodId int64) (*proto.GoodsDetail, error) {
	data, err := mysql.GetGoodsDetail(ctx, goodId)
	if err != nil {
		return nil, err
	}

	resp := &proto.GoodsDetail{
		GoodsId: data.GoodsId,

		CategoryId:  data.CategoryId,
		Status:      int32(data.Status),
		Title:       data.Title,
		MarketPrice: fmt.Sprintf("%.2f", float64(data.MarketPrice/100)),
		Price:       fmt.Sprintf("%.2f", float64(data.Price/100)),
		Brief:       data.Brief,
	}
	return resp, nil
}

func GetGoodsByRoomId(ctx context.Context, roomId int64) (*proto.GoodsListResp, error) {
	// 查询出roomId下所有的goodsId
	objList, err := mysql.GetGoodsByRoomId(ctx, roomId)
	if err != nil {
		return nil, err
	}

	// 拿出所有商品Id
	// 记住当前讲解Id

	var (
		currGoodsId int64
		idList      = make([]int64, 0, len(objList))
	)

	for _, obj := range objList {
		idList = append(idList, obj.GoodsId)
		if obj.IsCurrent == 1 {
			currGoodsId = obj.GoodsId
		}
	}

	goodsList, err := mysql.GetGoodsById(ctx, idList)
	if err != nil {
		return nil, err
	}

	//拼装响应数据
	data := make([]*proto.GoodsInfo, 0, len(goodsList))
	for _, goods := range goodsList {
		var headImgs []string
		json.Unmarshal([]byte(goods.HeadImgs), &headImgs)
		data = append(data, &proto.GoodsInfo{
			GoodsId:     goods.GoodsId,
			CategoryId:  goods.CategoryId,
			Status:      int32(goods.Status),
			Title:       goods.Title,
			MarketPrice: fmt.Sprintf("%.2f", float64(goods.MarketPrice/100)),
			Price:       fmt.Sprintf("%.2f", float64(goods.Price/100)),
			Brief:       goods.Brief,
			HeadImgs:    headImgs,
		})
	}

	resp := &proto.GoodsListResp{
		CurrentGoodsId: currGoodsId,
		Data:           data,
	}
	return resp, nil

}
