package simulation

import "stock_simulate/results"

const (
	DownBuyDays = 4
	UpBuyPct    = 5 // 此处是指百分比，比如5是指5%
)

// 这个判定逻辑如下：
// 1. 首先就是连续下降N天(下降是指当天的收盘价比开盘价低)
// 2. 某天开始上涨，当天涨幅超过5%（当天买入）或者连续两条上涨（第二天买入）
func JudgeBuyTime(baseInfos []results.StockBaseInfo) []OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]OperateInfo, len(baseInfos))

	// 开始判定过程
	downDays := 0
	upDays := 0
	for i, info := range baseInfos {
		tempOpeInfo := OperateInfo{}
		if info.Close < info.Open {
			downDays += 1
			upDays = 0
		} else if info.Open > info.Close {
			upDays += 1
			downDays = 0
		} else {
			upDays = 0
			downDays = 0
		}

		// TODO -- 此处是错误的吧？
		if downDays >= DownBuyDays && info.PctChg > UpBuyPct {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.1
		} else if downDays >= DownBuyDays && (i + 1) < len(baseInfos) && baseInfos[i + 1].Close > baseInfos[i + 1].Open {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.1
		} else {
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
		}
		retVal[i] = tempOpeInfo
	}
	return retVal
}
