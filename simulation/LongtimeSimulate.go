package simulation

import (
	"stock_simulate/datacenter"
	"stock_simulate/results"
)

const (
	LongTimeTargetWin = 0.4 // 目标盈利百分比
	NoWinDays         = 30  // 到达指定天数仍然没有盈利，就卖出操作（全仓）
	EMAUpDays         = 20  // EMA连续四天上涨才开始买入
)

// 根据EMA来进行判定
// 1. 如果EMA60开始上涨，那么就买入
// 2. 如果EMA60开始下跌，那么就卖出（相比与前一天下跌了）（全仓）
// 3. 如果达到指定天数之后仍然没有达到指定盈利，那么也卖出（全仓）（或者换种策略，持有就持有，当稍微有盈利了之后才卖出呢？）
func LongEmaSimulate(baseInfos []results.StockBaseInfo) []OperateInfo {
	if baseInfos == nil || len(baseInfos) == 0 {
		return nil
	}
	retVal := make([]OperateInfo, len(baseInfos))

	// 查询EMA相关的值，已经计算好了存储在数据库里面了
	var infos []results.EMAValue
	dataCenter := datacenter.GetInstance()
	tsCode := baseInfos[0].TsCode
	beginTradeDate := baseInfos[0].TradeDate
	sql := "select ts_code, trade_date, ifnull(ema_60, 0) ema_60, ifnull(ema_15, 0) ema_15, ifnull(ema_5, 0) ema_5 from ema_value where ts_code='" + tsCode + "' and trade_date>='" + beginTradeDate + "' order by trade_date"
	err := dataCenter.Db.Select(&infos, sql)
	if err != nil {
		panic(err)
		//return retVal
	}

	emaUpDays := 0
	for i, emaValue := range infos {
		if i > 0 && emaValue.EMA60 > infos[i-1].EMA60 {
			emaUpDays = emaUpDays + 1
		}
		if i < EMAUpDays {
			tempOpeInfo := OperateInfo{}
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
			retVal[i] = tempOpeInfo
			continue
		}

		tempOpeInfo := OperateInfo{}
		if emaUpDays > EMAUpDays && emaValue.EMA5 > infos[i-1].EMA5 {
			tempOpeInfo.OpeFlag = BuyFlag
			tempOpeInfo.OpePercent = 0.5
			// FIXME -- 增加了一个判定条件，如果是中等程度地EMA开始下降的话，我们就开始卖出好了
		} else if emaValue.EMA60 < infos[i-1].EMA60 || emaValue.EMA15 < infos[i-1].EMA15 {
			tempOpeInfo.OpeFlag = SoldFlag
			tempOpeInfo.OpePercent = 1
		} else {
			tempOpeInfo.OpeFlag = Nothing
			tempOpeInfo.OpePercent = 0
		}
		retVal[i] = tempOpeInfo
	}
	return retVal
}
