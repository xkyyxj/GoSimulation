package findTarget

import "stock_simulate/results"

const (
	UpJudgeDays = 2
)

func judgeIsUpSignal(baseInfos []results.StockBaseInfo) bool {
	if len(baseInfos) < UpJudgeDays {
		return false
	}

	retRst := true
	for index, item := range baseInfos {
		if index >= UpJudgeDays {
			return retRst
		}

		// 判定是否两天上涨
		retRst = retRst && item.PctChg > 0
	}
	return retRst
}
