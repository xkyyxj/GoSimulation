package simulation

//
//import (
//	"stock_simulate/datacenter"
//	"stock_simulate/results"
//	"strconv"
//)
//
//const (
//	HistoryDownConsiderBuy = 200 // 200天历史低值的话考虑买入
//	DownEmaField           = 4
//	DownEmaSoldDays        = 4 // 当EMA三天下降的时候就卖出好吧
//	DownSoldPct			   = -0.8
//	DownPctSoldPct         = 0.7	// 加个比上次买入下跌到了DownSoldPct时卖出的百分比
//
//	LastWin					= 1	// 上次获利
//	LastLost 				= 2	// 上次失利
//	LastNothing				= 3	// 上次竹篮打水一场空，既没有获利也没有失利
//)
//
//// 历史低值验证方法
//// 1. 当当天价格达到了HistoryDownConsiderBuy天的历史低值时，并且第二天价格开始突破的时候考虑买入
//// 2. 结合EMA吧，当EMA4三天下降的时候卖出
//func HistoryDownJudge(baseInfos []results.StockBaseInfo) []OperateInfo {
//	if baseInfos == nil || len(baseInfos) == 0 {
//		return nil
//	}
//	retVal := make([]OperateInfo, len(baseInfos))
//
//	startIndex := 0 // 开始写入买卖信息的天
//	// 重新查询一边基础信息，历史长一点
//	var preInfos []results.StockBaseInfo
//	beginDate := baseInfos[0].TradeDate
//	dataCenter := datacenter.GetInstance()
//	sql := "select * from stock_base_info where trade_date < '" + beginDate + "' and ts_code='" + baseInfos[0].TsCode
//	sql += "' limit " + strconv.Itoa(HistoryDownConsiderBuy)
//	_ = dataCenter.Db.Select(&preInfos, sql)
//	if preInfos != nil && len(preInfos) > 0 {
//		startIndex = len(preInfos)
//		baseInfos = append(preInfos, baseInfos...)
//	}
//
//	// 查询下对应的EMA数据
//	var infos []results.EMAValue
//	tsCode := baseInfos[0].TsCode
//	sql = "select ts_code, trade_date, ifnull(ema_4, 0) ema_4 from ema_value where ts_code='" + tsCode + "' and trade_date>='" + beginDate + "' order by trade_date"
//	err := dataCenter.Db.Select(&infos, sql)
//	if err != nil {
//		panic(err)
//		//return retVal
//	}
//
//	// TODO -- 有的股票没有ema_value，咋回事？先跳过去吧
//	if infos == nil || len(infos) == 0 {
//		return retVal
//	}
//
//	downDaysCount := 0
//	//var lastBuyPrice float64 = 0
//	//lastOpeRst := LastNothing
//	for i, value := range baseInfos {
//		if i < startIndex-1 {
//			continue
//		} else if i == startIndex-1 {
//			downDaysCount = judgeIsMinPrice(baseInfos, i, value.Close, downDaysCount)
//			continue
//		}
//
//		emaStartIndex := i - DownEmaSoldDays - startIndex
//		if emaStartIndex < 0 {
//			continue
//		}
//
//		hasBuy := false
//		tempOpeInfo := OperateInfo{}
//		if downDaysCount >= HistoryDownConsiderBuy {
//			// 此处意味着前一天的价格已经是HistoryDownConsiderBuy天的最低值了
//			// 判定下是否可以买入，也就是当天的收盘价是否比昨天的收盘价更高，同时EMA开始上涨了
//			// FIXME -- EMA值大于0？？这是什么神仙判定逻辑？？？？？
//			if value.PctChg > 0 && infos[emaStartIndex].EMA4 > 0 {
//				tempOpeInfo.OpeFlag = BuyFlag
//				tempOpeInfo.OpePercent = 0.3
//				//lastBuyPrice = value.Close
//				hasBuy = true
//			}
//		}
//		downDaysCount = judgeIsMinPrice(baseInfos, i, value.Close, downDaysCount)
//
//		if hasBuy {
//			retVal[i-startIndex] = tempOpeInfo
//			continue
//		}
//		// 根据EMA的值判定下是否需要卖出
//		alwaysDown := true
//		if len(infos) == 0 {
//			println("ts code is " + tsCode)
//		}
//		opePct := 0.4
//		preEmaValue := infos[emaStartIndex].EMA4
//		for emaStartIndex++; emaStartIndex <= i-startIndex && alwaysDown; emaStartIndex++ {
//			// 如果持续下降或者是加个降低到一定的百分比，那么也卖出
//			//downPct := (value.Close - lastBuyPrice) / lastBuyPrice
//			//if downPct < DownSoldPct {
//			//	// FIXME -- 写死一个数值，后续可以修正下
//			//	opePct = DownPctSoldPct
//			//	break
//			//}
//			if infos[emaStartIndex].EMA4 <= preEmaValue {
//				preEmaValue = infos[emaStartIndex].EMA4
//			} else {
//				alwaysDown = false
//			}
//		}
//
//		if alwaysDown {
//			tempOpeInfo.OpeFlag = SoldFlag
//			tempOpeInfo.OpePercent = opePct
//		} else {
//			tempOpeInfo.OpeFlag = Nothing
//			tempOpeInfo.OpePercent = 0
//		}
//		retVal[i-startIndex] = tempOpeInfo
//		downDaysCount = judgeIsMinPrice(baseInfos, i, value.Close, downDaysCount)
//	}
//
//	return retVal
//}
//
//func judgeIsMinPrice(baseInfos []results.StockBaseInfo, i int, currClose float64, downDaysCount int) int {
//	// 暴力法判定当前价格是不是最小值
//	searchBegin := i - HistoryDownConsiderBuy - 50
//	if searchBegin < 0 {
//		searchBegin = 0
//	}
//	for j := searchBegin; j < i; j++ {
//		if baseInfos[j].Close > currClose {
//			downDaysCount += 1
//		} else {
//			downDaysCount = 0
//			break
//		}
//	}
//	return downDaysCount
//}
