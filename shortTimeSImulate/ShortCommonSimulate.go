package shortTimeSImulate

import (
	"io/ioutil"
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"stock_simulate/simulation"
	"strconv"
	"sync"
)

const (
	SoldDays = 5 // 达到了多少天之后就卖出操作，短期操作
)

var mutex sync.Mutex

func Simulate(dirName string) {
	dataCenter := datacenter.GetInstance()
	var currIndex int
	currIndex = 0
	stockList := dataCenter.QueryStockCodes("")
	channelSlice := make([]<-chan simulation.SimulateRst, simulation.DefaultThreadNum)
	var waitGroup sync.WaitGroup
	waitGroup.Add(simulation.DefaultThreadNum)
	for i := 0; i < simulation.DefaultThreadNum; i++ {
		channel := make(chan simulation.SimulateRst, 10)
		go singleSimulate(&currIndex, stockList, channel, &waitGroup, dirName)
		channelSlice[i] = channel
	}

	// 最终结果统计
	waitGroup.Wait()
	println("in here!!")
	finalRst := simulation.SimulateRst{
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

func singleSimulate(index *int, stockList []string, channel chan simulation.SimulateRst, waitGroup *sync.WaitGroup, dirName string) {
	defer waitGroup.Done()
	// 最终返回结果
	simulateRst := simulation.SimulateRst{
		WinNum:       0,
		LostNum:      0,
		MaxWinPct:    0,
		MaxLostPct:   0,
		MaxWinStock:  "",
		MaxLostStock: "",
		Parameter:    "",
	}
	for {
		// 创建初始持仓信息
		holdInfos := simulation.StockHoldInfo{
			InitMny: simulation.InitMny,
			HoldNum: 0,
			LeftMny: simulation.InitMny,
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

		// 记录买入信息
		var timeIndexBuyInfo []*simulation.OperationDetail
		var buyPriceOrderInfo []*simulation.OperationDetail

		// 查询出对应的股票基本信息来
		dataCenter := datacenter.GetInstance()
		baseInfos := dataCenter.QueryStockBaseInfo(" ts_code='" + tsCode + "' order by trade_date desc limit 1200")
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
		retOpeTime := FourDayDownJudge(baseInfos)
		// 开始做分析
		var lastDetail simulation.OperationDetail
		lastDetail.TotalMny = simulation.InitMny
		for i, info := range retOpeTime {
			tempDetail := simulation.OperationDetail{}
			tempOpePct := info.OpePercent
			if info.OpeFlag == simulation.BuyFlag {
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
				tempDetail.OpeFlag = simulation.BuyDisplay
				tempDetail.TradeIndex = i
				tempDetail.AddDetailToExcelData(&excelData)
				timeIndexBuyInfo = append(timeIndexBuyInfo, &tempDetail)
				buyPriceOrderInfo = append(buyPriceOrderInfo, &tempDetail)

				// 针对买入价格做下排序
				for j := len(buyPriceOrderInfo) - 1; j > 0; j-- {
					if buyPriceOrderInfo[j].OpeClose > buyPriceOrderInfo[j-1].OpeClose {
						temp := buyPriceOrderInfo[j]
						buyPriceOrderInfo[j] = buyPriceOrderInfo[j-1]
						buyPriceOrderInfo[j-1] = temp
					} else {
						break
					}
				}
				lastDetail = tempDetail
			}
			tempOpeNum := 0
			// 检查下是否有过期的买入记录
			for {
				if len(timeIndexBuyInfo) <= 0 {
					break
				}
				// 已经卖出的去掉
				if timeIndexBuyInfo[0].HasSold {
					timeIndexBuyInfo = timeIndexBuyInfo[1:]
					continue
				}
				// FIXME -- 此处指定了买入的必须盈利了才能卖出
				// && baseInfos[i].Close > timeIndexBuyInfo[0].OpeClose
				if len(timeIndexBuyInfo) > 0 && (i-timeIndexBuyInfo[0].TradeIndex) > SoldDays {
					tempOpeNum += timeIndexBuyInfo[0].OpeNum
					timeIndexBuyInfo[0].HasSold = true
					timeIndexBuyInfo = timeIndexBuyInfo[1:]
				} else {
					break
				}
			}

			// 卖出都是按手为单位(100的整数)
			realOpeNum := int(tempOpeNum) - (int(tempOpeNum) % 100)
			if realOpeNum == 0 {
				continue
			}
			// 更新持仓信息
			holdInfos.HoldNum -= realOpeNum
			holdInfos.LeftMny += float64(realOpeNum) * baseInfos[i].Close

			// 更新操作信息
			soldDetail := simulation.OperationDetail{}
			soldDetail.TsCode = tsCode
			soldDetail.OpeClose = baseInfos[i].Close
			soldDetail.OpeNum = realOpeNum
			soldDetail.TradeDate = baseInfos[i].TradeDate
			soldDetail.HoldNum = holdInfos.HoldNum
			soldDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
			soldDetail.LeftMny = holdInfos.LeftMny
			soldDetail.TotalMny = soldDetail.HoldMny + soldDetail.LeftMny
			soldDetail.OpeFlag = simulation.SoldDisplay
			soldDetail.AddDetailToExcelData(&excelData)
			lastDetail = soldDetail
		}
		if lastDetail.TotalMny > simulation.InitMny {
			simulateRst.WinNum += 1
			winPct := (lastDetail.TotalMny - simulation.InitMny) / simulation.InitMny
			if winPct > simulateRst.MaxWinPct {
				simulateRst.MaxWinPct = winPct
			}
		} else {
			simulateRst.LostNum += 1
			winPct := (simulation.InitMny - lastDetail.TotalMny) / simulation.InitMny
			if winPct > simulateRst.MaxLostPct {
				simulateRst.MaxLostPct = winPct
			}
		}

		// 将实时分析结果写入到Excel文件当中去
		fileName := tsCode + ".xlsx"
		excelWriter := file.New(fileName, dirName)
		excelWriter.Write(excelData)
	}

	print("max lost pct is " + strconv.Itoa(int(simulateRst.MaxLostPct)))
	channel <- simulateRst
}
