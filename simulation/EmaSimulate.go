package simulation

import (
	"stock_simulate/datacenter"
	"stock_simulate/results"
)

const (
	EMAField        = 3
	EMAUpBuyCount   = 3
	EMADownSoldDays = 4 // 到达多少天的下降了考虑下卖出
	EWADownSoldPct  = -0.04
)

// 根据EMA来进行判定
// 1. 如果EMA3连续EMAUpBuyCount天上涨就买入
// 2. 如果EMA3连续EMADownSoldDays天下跌就卖出
func EMAJudgeBuyTime(baseInfos []results.StockBaseInfo) []OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]OperateInfo, len(baseInfos))

	// 查询EMA相关的值，已经计算好了存储在数据库里面了
	var infos []results.EMAValue
	dataCenter := datacenter.GetInstance()
	tsCode := baseInfos[0].TsCode
	beginTradeDate := baseInfos[0].TradeDate
	sql := "select ts_code, trade_date, ifnull(ema_3, 0) ema_3 from ema_value where ts_code='" + tsCode + "' and trade_date>='" + beginTradeDate + "' order by trade_date"
	err := dataCenter.Db.Select(&infos, sql)
	if err != nil {
		panic(err)
		//return retVal
	}

	upDays := 0
	downDays := 0
	var lastBuyPrice float64 = 0
	for i, emaValue := range infos {
		if i == 0 {
			tempOpeInfo := OperateInfo{}
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
			retVal[i] = tempOpeInfo
			continue
		}

		if emaValue.EMA3 > infos[i-1].EMA3 {
			upDays += 1
			downDays = 0
		} else {
			upDays = 0
			downDays += 1
		}

		// 相比于上一次的变动百分比
		var changePct float64 = 0
		if lastBuyPrice != 0 {
			changePct = (baseInfos[i].Close - lastBuyPrice) / lastBuyPrice
		}
		tempOpeInfo := OperateInfo{}
		if upDays >= EMAUpBuyCount {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.2
			lastBuyPrice = baseInfos[i].Close
		} else if downDays >= EMADownSoldDays && changePct <= EWADownSoldPct {
			// 到达了下降天数之后，还必须到达指定的下降百分比
			tempOpeInfo.OpeFlag = SoldFlag
			tempOpeInfo.OpePercent = 0.3
		} else {
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
		}
		retVal[i] = tempOpeInfo
	}
	return retVal
}
