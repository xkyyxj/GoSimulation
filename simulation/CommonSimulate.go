package simulation

import (
	"io/ioutil"
	"math"
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"stock_simulate/results"
	"sync"
)

const (
	InitMny          = 100000
	DefaultThreadNum = 1

	DisplayDeltaPct = 0.03 // 当当前持仓金额相比于上次持仓金额的百分比达到该数值时，即便无操作也将该条结果显示到最终的交易明细当中
)

var mutex sync.Mutex

func Simulate(dirName string) {
	dataCenter := datacenter.GetInstance()
	var currIndex int
	currIndex = 0
	stockList := dataCenter.QueryStockCodes(" ts_code = '601100.SH'")
	channelSlice := make([]<-chan SimulateRst, DefaultThreadNum)
	var waitGroup sync.WaitGroup
	waitGroup.Add(DefaultThreadNum)
	for i := 0; i < DefaultThreadNum; i++ {
		channel := make(chan SimulateRst, 10)
		go simulateGrp(&currIndex, stockList, channel, &waitGroup, dirName)
		channelSlice[i] = channel
	}

	// 最终结果统计
	waitGroup.Wait()
	println("in here!!")
	finalRst := SimulateRst{
		WinNum:     0,
		LostNum:    0,
		MaxWinPct:  0,
		MaxLostPct: 0,
	}
	for _, channel := range channelSlice {
		tempVal := <-channel
		finalRst.WinNum += tempVal.WinNum
		finalRst.LostNum += tempVal.LostNum
		if finalRst.MaxWinPct < tempVal.MaxWinPct {
			finalRst.MaxWinPct = tempVal.MaxWinPct
		}
		if finalRst.MaxLostPct < tempVal.MaxLostPct {
			finalRst.MaxLostPct = tempVal.MaxLostPct
		}
	}

	finalRstString := finalRst.ToString()
	filePath := file.DefaultPreWorkspace + "\\" + dirName + "\\sum.txt"
	_ = ioutil.WriteFile(filePath, []byte(finalRstString), 0777) //如果文件a.txt已经存在那么会忽略权限参数，清空文件内容。文件不存在会创建文件赋予权限
	println("Calculate finished!!!")
}

func simulateGrp(index *int, stockList []string, channel chan SimulateRst, waitGroup *sync.WaitGroup, dirName string) {
	defer waitGroup.Done()
	// 最终返回结果
	simulateRst := SimulateRst{}
	for {
		// 创建初始持仓信息
		holdInfos := StockHoldInfo{
			InitMny: InitMny,
			HoldNum: 0,
			LeftMny: InitMny,
		}
		// 创建写入Excel的数据
		excelData := file.ExcelData{
			Data: make(map[string][]interface{}),
		}
		// 创建改制股票
		mutex.Lock()
		if *index >= len(stockList) {
			println("Group Calculate finished!!")
			mutex.Unlock()
			break
		}
		tsCode := stockList[*index]
		*index = *index + 1
		mutex.Unlock()

		// 查询出对应的股票基本信息来
		dataCenter := datacenter.GetInstance()
		baseInfos := dataCenter.QueryStockBaseInfo(" ts_code='" + tsCode + "' order by trade_date desc limit 1000")
		if baseInfos == nil || len(baseInfos) == 0 {
			continue
		}

		// 对切片进行反序操作
		for i, j := 0, len(baseInfos)-1; i < j; i, j = i+1, j-1 {
			baseInfos[i], baseInfos[j] = baseInfos[j], baseInfos[i]
		}
		//retOpeTime := DayJudgeBuyTime(baseInfos)
		//retOpeTime := EMAJudgeBuyTime(baseInfos)
		//retOpeTime := HistoryDownJudge(baseInfos)
		//retOpeTime := HistoryDownLongJudge(baseInfos)
		retOpeTime := LongEmaSimulate(baseInfos)
		// 开始做分析
		var lastDetail OperationDetail
		for i, info := range retOpeTime {
			tempDetail := OperationDetail{}
			tempOpePct := info.OpePercent
			if info.OpeFlag == BuyFlag {
				tempBuyMny := holdInfos.LeftMny * tempOpePct
				if tempBuyMny > holdInfos.LeftMny {
					tempBuyMny = holdInfos.LeftMny
				}
				// 买入都是按手为单位(100的整数)
				tempBuyNum := tempBuyMny / baseInfos[i].Close
				intBuyNum := int(tempBuyNum)
				intBuyNum = intBuyNum - (intBuyNum % 100)

				realBuyMny := float64(intBuyNum) * baseInfos[i].Close
				holdInfos.LeftMny = holdInfos.LeftMny - realBuyMny
				holdInfos.HoldNum += intBuyNum

				tempDetail.TsCode = tsCode
				tempDetail.OpeClose = baseInfos[i].Close
				tempDetail.OpeNum = intBuyNum
				tempDetail.TradeDate = baseInfos[i].TradeDate
				tempDetail.HoldNum = holdInfos.HoldNum
				tempDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
				tempDetail.LeftMny = holdInfos.LeftMny
				tempDetail.TotalMny = tempDetail.HoldMny + tempDetail.LeftMny
				tempDetail.OpeFlag = BuyDisplay
				tempDetail.AddDetailToExcelData(&excelData)
				lastDetail = tempDetail
			} else if info.OpeFlag == SoldFlag {
				tempOpeNum := float64(holdInfos.HoldNum) * tempOpePct
				// 卖出都是按手为单位(100的整数)
				realOpeNum := int(tempOpeNum) - (int(tempOpeNum) % 100)
				if realOpeNum == 0 {
					continue
				}
				// 更新持仓信息
				holdInfos.HoldNum -= realOpeNum
				holdInfos.LeftMny += float64(realOpeNum) * baseInfos[i].Close

				// 更新操作信息
				tempDetail.TsCode = tsCode
				tempDetail.OpeClose = baseInfos[i].Close
				tempDetail.OpeNum = realOpeNum
				tempDetail.TradeDate = baseInfos[i].TradeDate
				tempDetail.HoldNum = holdInfos.HoldNum
				tempDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
				tempDetail.LeftMny = holdInfos.LeftMny
				tempDetail.TotalMny = tempDetail.HoldMny + tempDetail.LeftMny
				tempDetail.OpeFlag = SoldDisplay
				tempDetail.AddDetailToExcelData(&excelData)
				lastDetail = tempDetail
			} else {
				addDeltaInfo(&lastDetail, &baseInfos[i], &excelData)
			}
		}
		if lastDetail.TotalMny > InitMny {
			simulateRst.WinNum += 1
			winPct := (lastDetail.TotalMny - InitMny) / InitMny
			if winPct > simulateRst.MaxWinPct {
				simulateRst.MaxWinPct = winPct
			}
		} else {
			simulateRst.LostNum += 1
			winPct := (InitMny - lastDetail.TotalMny) / InitMny
			if winPct > simulateRst.MaxLostPct {
				simulateRst.MaxLostPct = winPct
			}
		}

		// 将实时分析结果写入到Excel文件当中去
		fileName := tsCode + ".xlsx"
		excelWriter := file.New(fileName, dirName)
		excelWriter.Write(excelData)
	}
	channel <- simulateRst
}

func addDeltaInfo(lastDetail *OperationDetail, baseInfo *results.StockBaseInfo, excelData *file.ExcelData) {
	detail := OperationDetail{
		TsCode:     lastDetail.TsCode,
		TsName:     lastDetail.TsName,
		OpeNum:     0,
		OpeClose:   baseInfo.Close,
		TradeDate:  baseInfo.TradeDate,
		HoldNum:    lastDetail.HoldNum,
		HoldMny:    0, // 下面做计算
		LeftMny:    lastDetail.LeftMny,
		OpeFlag:    NothingOpe,
		TradeIndex: 0,
		HasSold:    false,
	}
	currHoldMny := float64(lastDetail.HoldNum) * baseInfo.Close
	deltaMny := math.Abs(currHoldMny - lastDetail.HoldMny)
	deltaPct := deltaMny / lastDetail.HoldMny
	detail.HoldMny = currHoldMny
	detail.TotalMny = detail.HoldMny + detail.LeftMny
	if deltaPct >= DisplayDeltaPct {
		detail.AddDetailToExcelData(excelData)
	}
}
