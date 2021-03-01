package simulation

//
//import (
//	"stock_simulate/datacenter"
//	"stock_simulate/results"
//	"strconv"
//)
//
//const (
//	EmaField		= 4		// 该值指定的Ema开始上涨
//	VolField		= 4		// 该值指定的Vol移动平均值开始上涨
//	WinSoldPct		= 0.15	// 盈利达到多少百分比就卖出
//)
//
//// 成交量以及价格EMA方法验证，齐头并进方可入场
//// 1.
//func VolEmaJudge(baseInfos []results.StockBaseInfo) []results.OperateInfo {
//	if baseInfos == nil || len(baseInfos) == 0 {
//		return nil
//	}
//	retVal := make([]results.OperateInfo, len(baseInfos))
//
//	// 第一步：查询对应的Ema值
//	var closeEma []results.EMAValue
//	emaField := "ema_" + strconv.Itoa(EmaField)
//	sql := "select ema_" + emaField + " from ema_value where ts_code='" + baseInfos[0].TsCode + "' and trade_date>='" + baseInfos[0].TradeDate + "'"
//	_ = datacenter.GetInstance().Db.Select(&closeEma, sql)
//	if len(closeEma) == 0 {
//		return nil
//	}
//
//	// 第二步：查询对应的VOL值
//	var volEma []float64
//	volField := strconv.Itoa(VolField)
//	sql = "select ema_" + volField + " from vol_ema where ts_code='" + baseInfos[0].TsCode + "' and trade_date>='" + baseInfos[0].TradeDate + "'"
//	datacenter.GetInstance().Db.Select(&volEma, sql)
//	if len(volEma) == 0 {
//		return nil
//	}
//	var lastBuyPrice float64
//	lastBuyIndex := 0
//
//	// 填充第一条数据
//	retVal[0] = results.OperateInfo{
//		OpeFlag: results.Nothing,
//	}
//
//	for i, baseInfo := range baseInfos {
//		if i == 0 {
//			continue
//		}
//
//		canContinue := true
//		// 第一步：判定ema是否上涨
//		canContinue = canContinue && closeEma[i].EMA4 <= closeEma[i - 1].EMA4
//
//		// 第二部：判定vol_ema是否上涨
//		canContinue = canContinue && volEma[i] <= volEma[i - 1]
//
//		var opeInfo results.OperateInfo
//		if canContinue {
//			opeInfo.OpePercent = 0.1
//			opeInfo.OpeFlag = results.BuyFlag
//			lastBuyPrice = baseInfo.Close
//			lastBuyIndex
//		}
//
//		// 卖出判定
//		// 盈利百分比
//		winPct := (baseInfo.Close - lastBuyPrice) / lastBuyPrice
//		if winPct >
//
//		retVal[i] = opeInfo
//	}
//	return retVal
//}
