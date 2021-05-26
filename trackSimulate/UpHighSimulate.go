package trackSimulate

import (
	"stock_simulate/datacenter"
	"stock_simulate/results"
)

const ()

// 判定是否持续性上涨，对于持续性上涨的做买入操作
// 卖出操作
func UpHighSimulate(baseInfos []results.StockBaseInfo) []OperateInfo {
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
