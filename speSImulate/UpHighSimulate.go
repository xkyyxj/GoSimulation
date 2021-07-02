package speSImulate

import (
	"stock_simulate/results"
	"stock_simulate/simulation"
)

const (
	UpHighCount = 10  // 连续上涨天数
	UpHighPct   = 0.2 // 连续上涨百分比
)

// 判定是否开启上涨模式，对于开启上涨的做买入操作
// 持续性上涨的判定条件很简单：1.在UpHighCount天之内上涨了UpHighPct的百分比
// 卖出操作
func UpHighSimulate(baseInfos []results.StockBaseInfo) []simulation.OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]simulation.OperateInfo, len(baseInfos))

	// 判定是否上涨
	for i, _ := range baseInfos {
		if i < UpHighCount {
			tempOpeInfo := simulation.OperateInfo{}
			tempOpeInfo.OpeFlag = simulation.Nothing
			tempOpeInfo.OpePercent = 0
			retVal[i] = tempOpeInfo
			continue
		}

		// 前UpCount天的股票走势
		firstClose := baseInfos[i-UpHighCount].Close
		lastClose := baseInfos[i].Close
		tempUpPct := (lastClose - firstClose) / firstClose
		if tempUpPct >= UpHighPct {
			tempOpeInfo := simulation.OperateInfo{}
			tempOpeInfo.OpeFlag = simulation.BuyFlag
			tempOpeInfo.OpePercent = 0.5
			retVal[i] = tempOpeInfo
		} else if baseInfos[i].PctChg > 0 {
			// Important -- 主要考虑这种情况：以前买入的失利了，现在开始上涨了，name今天可以持续持有以观后续，
			// 避免损失过大
			tempOpeInfo := simulation.OperateInfo{}
			tempOpeInfo.OpeFlag = simulation.HoldFlag
			tempOpeInfo.OpePercent = 0
			retVal[i] = tempOpeInfo
		} else {
			tempOpeInfo := simulation.OperateInfo{}
			tempOpeInfo.OpeFlag = simulation.Nothing
			tempOpeInfo.OpePercent = 0
			retVal[i] = tempOpeInfo
		}
		// TODO -- 是不是还可以考虑这么一种情况：以前失利了，但是最近上涨了一波后，又开始下跌了
	}
	return retVal
}
