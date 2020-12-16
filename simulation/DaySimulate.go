package simulation

import "stock_simulate/results"

const (
	DownBuyDays  = 4
	DownSoldDays = 3
	UpBuyPct     = 5 // 此处是指百分比，比如5是指5%
)

// 这个判定逻辑如下：
// 1. 首先就是连续下降N天(下降是指当天的收盘价比开盘价低)
// 2. 某天开始上涨，当天涨幅超过5%（当天买入）或者连续两条上涨（第二天买入）
// 3. 如果当天的下降幅度超过了5%，那么卖出
func DayJudgeBuyTime(baseInfos []results.StockBaseInfo) []OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]OperateInfo, len(baseInfos))

	// 开始判定过程
	downDays := 0
	upDays := 0
	for i, info := range baseInfos {
		tempOpeInfo := OperateInfo{}

		// TODO -- 此处是错误的吧？
		if downDays >= DownBuyDays && info.PctChg > UpBuyPct {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.1
		} else if downDays >= DownBuyDays && (i+1) < len(baseInfos) && baseInfos[i+1].Close > baseInfos[i+1].Open {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.1
		} else if downDays >= DownSoldDays && baseInfos[i].Close < baseInfos[i].Open {
			tempOpeInfo.OpeFlag = SoldFlag
			// FIXME -- 此处写死了一个值，应该可以动态判定的
			tempOpeInfo.OpePercent = 0.3
		} else {
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
		}
		retVal[i] = tempOpeInfo

		if info.Close < info.Open || info.PctChg < 0 {
			// 当天的收盘价比较开盘价低或者当天收盘价比前一天的收盘价低
			downDays += 1
			upDays = 0
		} else if info.Open > info.Close {
			upDays += 1
			downDays = 0
		} else {
			upDays = 0
			downDays = 0
		}
	}
	return retVal
}
