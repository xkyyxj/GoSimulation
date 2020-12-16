package simulation

import (
	"io/ioutil"
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"sync"
)

const (
	InitMny          = 100000
	DefaultThreadNum = 100
)

var mutex sync.Mutex

func Simulate(dirName string) {
	dataCenter := datacenter.GetInstance()
	var currIndex int
	currIndex = 0
	stockList := dataCenter.QueryStockCodes("")
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
		winNum:     0,
		lostNum:    0,
		maxWinPct:  0,
		maxLostPct: 0,
	}
	for _, channel := range channelSlice {
		tempVal := <-channel
		finalRst.winNum += tempVal.winNum
		finalRst.lostNum += tempVal.lostNum
		if finalRst.maxWinPct < tempVal.maxWinPct {
			finalRst.maxWinPct = tempVal.maxWinPct
		}
		if finalRst.maxLostPct < tempVal.maxLostPct {
			finalRst.maxLostPct = tempVal.maxLostPct
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
		retOpeTime := HistoryDownJudge(baseInfos)
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
			}
		}
		if lastDetail.TotalMny > InitMny {
			simulateRst.winNum += 1
			winPct := (lastDetail.TotalMny - InitMny) / InitMny
			if winPct > simulateRst.maxWinPct {
				simulateRst.maxWinPct = winPct
			}
		} else {
			simulateRst.lostNum += 1
			winPct := (InitMny - lastDetail.TotalMny) / InitMny
			if winPct > simulateRst.maxLostPct {
				simulateRst.maxLostPct = winPct
			}
		}

		// 将实时分析结果写入到Excel文件当中去
		fileName := tsCode + ".xlsx"
		excelWriter := file.New(fileName, dirName)
		excelWriter.Write(excelData)
	}
	channel <- simulateRst
}
