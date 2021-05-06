package shortTimeSImulate

import (
	"stock_simulate/results"
	"stock_simulate/simulation"
)

const (
	DownDays = 4 // 连续下跌多少天之后考虑买入
)

// 提升短期买入盈利比率的判定函数
// 1. 当当天价格达到了HistoryDownConsiderBuy天的历史低值时，并且第二天价格开始突破的时候考虑买入
// 2. 结合EMA吧，当EMA4三天下降的时候卖出
func FourDayDownJudge(baseInfos []results.StockBaseInfo) []simulation.OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]simulation.OperateInfo, len(baseInfos))

	downDays := 0
	var preClose float64
	preClose = baseInfos[0].Close
	i := 1
	for ; i < DownDays; i++ {
		if baseInfos[i].Close < preClose {
			downDays += 1
			preClose = baseInfos[i].Close
		}

		tempOpeInfo := simulation.OperateInfo{}
		tempOpeInfo.OpeFlag = simulation.Nothing
		retVal[i] = tempOpeInfo
	}
	for ; i < len(baseInfos); i++ {
		tempOpeInfo := simulation.OperateInfo{}
		if downDays >= DownDays {
			tempOpeInfo.OpeFlag = simulation.BuyFlag
			tempOpeInfo.OpePercent = 0.1
		} else {
			tempOpeInfo.OpeFlag = simulation.Nothing
		}
		retVal[i] = tempOpeInfo

		if baseInfos[i].Close <= preClose {
			downDays += 1
			preClose = baseInfos[i].Close
		} else {
			downDays = 1
		}
	}

	return retVal
}
