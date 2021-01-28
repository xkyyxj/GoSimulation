package simulation

import (
	"stock_simulate/datacenter"
	"stock_simulate/results"
	"strconv"
)

const (
	TargetWinPct     = 0.8
	LongTimeHoldDays = 150 // 长期持有的话，持有天数，到期没有足够获利的话卖出操作
)

// 历史低值股票，长期持有看什么情况
func HistoryDownLongJudge(baseInfos []results.StockBaseInfo) []OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]OperateInfo, len(baseInfos))

	startIndex := 0 // 开始写入买卖信息的天
	// 重新查询一边基础信息，历史长一点
	var preInfos []results.StockBaseInfo
	beginDate := baseInfos[0].TradeDate
	dataCenter := datacenter.GetInstance()
	sql := "select * from stock_base_info where trade_date < '" + beginDate + "' and ts_code='" + baseInfos[0].TsCode
	sql += "' limit " + strconv.Itoa(HistoryDownConsiderBuy)
	_ = dataCenter.Db.Select(&preInfos, sql)
	if preInfos != nil && len(preInfos) > 0 {
		startIndex = len(preInfos)
		baseInfos = append(preInfos, baseInfos...)
	}

	// 查询下对应的EMA数据
	beginDate = baseInfos[0].TradeDate
	var infos []results.EMAValue
	tsCode := baseInfos[0].TsCode
	sql = "select ts_code, trade_date, ifnull(ema_4, 0) ema_4, ifnull(ema_9, 0) ema_9 from ema_value where ts_code='" + tsCode + "' and trade_date>='" + beginDate + "' order by trade_date"
	err := dataCenter.Db.Select(&infos, sql)
	if err != nil {
		panic(err)
		//return retVal
	}

	// TODO -- 有的股票没有ema_value，咋回事？先跳过去吧
	if infos == nil || len(infos) == 0 {
		return retVal
	}

	downDaysCount := 0
	downHasBuy := false
	lastBuyIndex := 0
	for i, value := range baseInfos {
		if i < startIndex-1 {
			continue
		} else if i == startIndex-1 {
			downDaysCount = judgeIsMinPrice(baseInfos, i, value.Close, downDaysCount)
			continue
		}

		tempOpeInfo := OperateInfo{}
		if downDaysCount >= HistoryDownConsiderBuy {
			// 判定一下，如果是相比于上次的买入价格，价格更低了，那么我们就执行卖出操作
			// 此处意味着前一天的价格已经是HistoryDownConsiderBuy天的最低值了
			// 验证下是不是ema开始上涨了？
			if value.PctChg > 0 && i > 1 && infos[i].EMA9 > infos[i-1].EMA9 {
				tempOpeInfo.OpeFlag = BuyFlag
				tempOpeInfo.OpePercent = 0.2
				downHasBuy = true
				lastBuyIndex = i
			}
			retVal[i-startIndex] = tempOpeInfo
		}

		if downHasBuy || downDaysCount < HistoryDownConsiderBuy {
			downDaysCount = judgeIsMinPrice(baseInfos, i, value.Close, downDaysCount)
		}

		if downHasBuy {
			downHasBuy = false
			continue
		}

		// 直到有盈利了才卖出，或者达到了指定的卖出天数，实在是hold不住了
		//currWinPct := (value.Close - baseInfos[lastBuyIndex].Close) / baseInfos[lastBuyIndex].Close
		//buyTime, _ := time.Parse("01022006", baseInfos[i].TradeDate)
		//currTime, _ := time.Parse("01022006", value.TradeDate)
		//yearAveWin := currTime.Sub(buyTime).Hours() / 24
		//yearAveWin = (365 / yearAveWin) * currWinPct
		//// 盈利达到目标百分比，并且EMA开始连续三天下降(按照年化利率算，年化40%)
		//if yearAveWin > TargetWinPct && i > 1 && infos[i].EMA9 < infos[i - 1].EMA9 {
		//	tempOpeInfo.OpeFlag = SoldFlag
		//	tempOpeInfo.OpePercent = 0.9
		//} else {
		//	tempOpeInfo.OpeFlag = Nothing
		//	tempOpeInfo.OpePercent = 0.9
		//}

		// 如果相比于上次的买入价格有盈利，并且EMA开始走低，那么就直接卖出（再加上个）
		if value.Close > baseInfos[lastBuyIndex].Close && infos[i].EMA9 < infos[i-1].EMA9 {
			tempOpeInfo.OpeFlag = SoldFlag
			tempOpeInfo.OpePercent = 0.5
		} else {
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
		}
		retVal[i-startIndex] = tempOpeInfo
	}

	return retVal
}
