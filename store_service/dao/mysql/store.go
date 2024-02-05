package mysql

import (
	"context"
	"errors"
	"fmt"
	"store_service/dao/redis"
	"store_service/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetStoreByGoodsId(ctx context.Context, goodsId int64) (*model.Store, error) {
	var data model.Store
	err := db.WithContext(ctx).
		Model(&model.Store{}).
		Where("goods_id = ? ", goodsId).
		First(&data).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.New("Query is failed!")
	}
	return &data, nil
}

func SetStoreByGoodsId(ctx context.Context, goodsId, num int64) error {
	// 采用悲观锁实现
	var data model.Store

	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(ctx).
			Model(&model.Store{}).
			Where("goods_id = ? ", goodsId).
			First(&data).Error

		if err != nil {
			return err
		}

		err = tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&model.Store{}).
			Where("goods_id = ? ", data.GoodsId).
			Update("num", num).
			Error

		if err != nil {
			zap.L().Error("SetStoreByGoodsId save failed,", zap.Int64("goods_id", goodsId))
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

/* func ReduceStore(ctx context.Context, goodsId, num int64) (*model.Store, error) {
	// 采用悲观锁实现
	// UPDATE 加行锁
	var data model.Store
	db.Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&model.Store{}).
			First(&data).Error
		if err != nil {
			return err
		}

		if data.Num-num < 0 {
			return errors.New("库存不足")
		}

		data.Num -= num

		err = tx.WithContext(ctx).
			Model(&data).
			Update("num", data.Num).Error
		if err != nil {
			zap.L().Info("ReduceStore save failed", zap.Int64("goods_id", goodsId))
			return err
		}
		return nil
	})
	return &data, nil
} */

/* func ReduceStore(ctx context.Context, goodsId, num int64) (*model.Store, error) {
	// 采用乐观锁实现
	// version版本号实现
	var (
		data      model.Store
		retry     = 0
		isSuccess = false
	)

	for retry < 20 && !isSuccess {
		err := db.WithContext(ctx).
			Model(&model.Store{}).
			Where("goods_id = ? ", goodsId).
			First(&data).Error
		if err != nil {
			return nil, err
		}

		if data.Num-num < 0 {
			return nil, errors.New("库存不足")
		}

		data.Num -= num

		n := db.WithContext(ctx).
			Model(&model.Store{}).
			Where("goods_id = ? and version = ?", goodsId, data.Version).
			Updates(map[string]interface{}{
				"goods_id": data.GoodsId,
				"num":      data.Num,
				"version":  data.Version + 1,
			}).RowsAffected
		if n < 1 {
			// fmt.Printf("GoodsId: %d try to reducestore %d time\n", goodsId, retry)
			retry++
			continue
		}
		isSuccess = true
		break
	}

	if !isSuccess {
		return nil, errors.New("库存扣减失败！")
	}
	return &data, nil
} */

func ReduceStore(ctx context.Context, goodsId, num, orderId int64) (*model.Store, error) {
	var data model.Store

	mutexname := fmt.Sprintf("xx-store-%d", goodsId)
	mutex := redis.Rs.NewMutex(mutexname)
	if err := mutex.Lock(); err != nil {
		return nil, errors.New("Get Redisync Failed!")
	}
	defer mutex.Unlock()

	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(ctx).
			Model(&model.Store{}).
			First(&data).Error
		if err != nil {
			return err
		}

		if data.Num-num < 0 {
			return errors.New("库存不足")
		}

		data.Num -= num
		data.Lock += num

		err = tx.WithContext(ctx).
			Save(&data).
			Error
		if err != nil {
			zap.L().Info("ReduceStore save failed", zap.Int64("goods_id", goodsId))
			return err
		}

		// 创建库存记录表
		storeRecord := model.StoreRecord{
			OrderId: orderId,
			GoodsId: goodsId,
			Num:     num,
			Status:  1, // 预扣减
		}
		err = tx.WithContext(ctx).
			Model(&model.StoreRecord{}).
			Create(&storeRecord).Error
		if err != nil {
			zap.L().Error("create StoreRecord failed", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// RollbackStock 监听rocketmq消息进行库存回滚
func RollbackStockByMsg(ctx context.Context, data model.OrderGoodsStockInfo) error {
	// 先查询库存数据，需要放到事务操作中
	db.Transaction(func(tx *gorm.DB) error {
		var sr model.StoreRecord
		err := tx.WithContext(ctx).
			Model(&model.StoreRecord{}).
			Where("order_id = ? and goods_id = ? and status = 1", data.OrderId, data.GoodsId).
			First(&sr).Error
		// 没找到记录
		// 压根就没记录或者已经回滚过 不需要后续操作
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		if err != nil {
			zap.L().Error("query stock_record by order_id failed", zap.Error(err), zap.Int64("order_id", data.OrderId), zap.Int64("goods_id", data.GoodsId))
			return err
		}
		// 开始归还库存
		var s model.Store
		err = tx.WithContext(ctx).
			Model(&model.Store{}).
			Where("goods_id = ?", data.GoodsId).
			First(&s).Error
		if err != nil {
			zap.L().Error("query stock by goods_id failed", zap.Error(err), zap.Int64("goods_id", data.GoodsId))
			return err
		}
		s.Num += data.Num  // 库存加上
		s.Lock -= data.Num // 锁定的库存减掉
		if s.Lock < 0 {    // 预扣库存不能为负
			return errors.New("回滚库存失败")
		}
		err = tx.WithContext(ctx).Save(&s).Error
		if err != nil {
			zap.L().Warn("RollbackStock stock save failed", zap.Int64("goods_id", s.GoodsId), zap.Error(err))
			return err
		}
		// 将库存扣减记录的状态变更为已回滚
		sr.Status = 3
		err = tx.WithContext(ctx).Save(&sr).Error
		if err != nil {
			zap.L().Warn("RollbackStock stock_record save failed", zap.Int64("goods_id", s.GoodsId), zap.Error(err))
			return err
		}
		return nil
	})
	return nil
}
