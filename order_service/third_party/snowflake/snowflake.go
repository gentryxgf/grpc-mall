package snowflake

import (
	"errors"
	"time"

	sf "github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

const (
	_defaultStartTime = "2023-08-28"
)

var node *sf.Node

func Init(startTime string, machineId int64) (err error) {
	if machineId < 0 {
		return errors.New("snowflake need machineId")
	}

	if len(startTime) == 0 {
		startTime = _defaultStartTime
	}

	var st time.Time
	st, err = time.Parse("2006-01-02", startTime)
	if err != nil {
		return
	}

	sf.Epoch = st.UnixNano() / 1000000
	node, err = sf.NewNode(machineId)
	zap.L().Info("Init snowflow success!")
	return
}

func GenID() int64 {
	return node.Generate().Int64()
}

func GenIDStr() string {
	return node.Generate().String()
}
