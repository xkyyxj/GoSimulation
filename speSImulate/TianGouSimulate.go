package speSImulate

import (
	"io/ioutil"
	"math"
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"stock_simulate/results"
	"stock_simulate/simulation"
	"strconv"
	"sync"
)

const (
	InitMny          = 100000
	DefaultThreadNum = 100
	// 原先的默认值是15，现在改成5试下效果，对于LonggSimulate
	NoWinDays   = 5     // 到达指定天数仍然没有盈利，就卖出操作（全仓）
	TempWinPCt  = 0.05  // 暂时性盈利百分比，如果在NoWinDays天之内没有达到该值指定的盈利百分比，那么做卖出操作
	NoWinPct    = 0.08  // 每笔亏损达到多少的时候就会卖出，不管是否到了NoWinDays指定的天数
	LongBackPct = -0.08 // 最大回撤达到这个百分比的时候，就在最终的输出结果当中做一个反馈
	BackSoldPct = 0.25  // 从最大盈利百分比回撤了百分之多少之后就卖出操作，譬如最大盈利百分比是50%，这个值是0.5，那么当盈利百分比达到25%时执行卖出操作

	DisplayDeltaPct = 0.05 // 当当前持仓金额相比于上次持仓金额的百分比达到该数值时，即便无操作也将该条结果显示到最终的交易明细当中
)

var mutex sync.Mutex

// TODO -- 看下盈利比率的中位数是什么？？？
// 舔狗交易法，懂得都懂
// 1. 定期观察该只股票，如果是发出了买入信号的话，我们就做买入操作（此处理想的买入信号是：开始上涨趋势）
// 2. 如果我们判断错误该只股票最近并不想上涨（其实妹子只是空虚寂寞冷，并不想和你有交集），那么我们就识趣点，换下一家股票（此处只能是继续观察）
//    这里包含两种情况：1.买入后立刻下跌，2.或者买入后长期没有上涨到指定涨幅
// 3. 持续不断观察，当有新的买入信号发出时，我们就做出买入决定（不因妹子的一次拒绝就放弃，舔狗到底~）
func Simulate(dirName string) {
	dataCenter := datacenter.GetInstance()
	var currIndex int
	currIndex = 0
	stockList := dataCenter.QueryStockCodes("")
	channelSlice := make([]<-chan simulation.SimulateRst, DefaultThreadNum)
	var waitGroup sync.WaitGroup
	waitGroup.Add(DefaultThreadNum)
	for i := 0; i < DefaultThreadNum; i++ {
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
		DetailInfo: make([]simulation.SingleStockSimulateRst, 0),
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
		finalRst.DetailInfo = append(finalRst.DetailInfo, tempVal.DetailInfo...)
	}

	// 将每只股票的盈利信息写入到文件当中
	allStockListWinInfo := file.ExcelData{
		Data: make(map[string][]interface{}),
	}
	finalRst.ConvertSimulateRstToExcelData(&allStockListWinInfo)
	fileName := "all_list.xlsx"
	excelWriter := file.New(fileName, dirName)
	excelWriter.Write(allStockListWinInfo)

	finalRstString := finalRst.ToString()
	finalRstString += "\n total stock Number is : " + strconv.Itoa(len(stockList))
	filePath := file.DefaultPreWorkspace + "\\" + dirName + "\\sum.txt"
	_ = ioutil.WriteFile(filePath, []byte(finalRstString), 0777) //如果文件a.txt已经存在那么会忽略权限参数，清空文件内容。文件不存在会创建文件赋予权限
	println("Calculate finished!!!")
}

func singleSimulate(index *int, stockList []string, channel chan simulation.SimulateRst, waitGroup *sync.WaitGroup, dirName string) {
	defer waitGroup.Done()
	// 最终返回结果
	simulateRst := simulation.SimulateRst{}
	for {
		// 创建初始持仓信息
		holdInfos := simulation.StockHoldInfo{
			InitMny: InitMny,
			HoldNum: 0,
			LeftMny: InitMny,
		}
		// 创建写入Excel的数据
		excelData := file.ExcelData{
			Data: make(map[string][]interface{}),
		}
		// 该只股票模拟过程当中的统计信息
		statisticInfo := simulation.SingleStockSimulateRst{}
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
		baseInfos := dataCenter.QueryStockBaseInfo(" ts_code='" + tsCode + "' order by trade_date desc limit 1000")
		if baseInfos == nil || len(baseInfos) == 0 {
			continue
		}

		// 初始化一些统计信息相关字段
		statisticInfo.TsCode = baseInfos[0].TsCode
		statisticInfo.TsName = baseInfos[0].TsCode
		// 此处赋值一个默认值，为了保证判定不会错误
		statisticInfo.LowestMny = 1000000

		// 对切片进行反序操作
		for i, j := 0, len(baseInfos)-1; i < j; i, j = i+1, j-1 {
			baseInfos[i], baseInfos[j] = baseInfos[j], baseInfos[i]
		}
		//retOpeTime := DayJudgeBuyTime(baseInfos)
		//retOpeTime := EMAJudgeBuyTime(baseInfos)
		//retOpeTime := HistoryDownJudge(baseInfos)
		//retOpeTime := simulation.LongEmaSimulate(baseInfos)
		//retOpeTime := UpSignalSimulate(baseInfos)
		retOpeTime := UpHighSimulate(baseInfos)
		// 开始做分析
		//lostCount := 0		// 失利次数
		var lastDetail simulation.OperationDetail
		lastDetail.LeftMny = InitMny
		lastDetail.TotalMny = InitMny
		for i, info := range retOpeTime {
			// 每次开始之前需要检查一下是不是达到了最大回撤的百分比
			tempHoldMny := float64(holdInfos.HoldNum) * baseInfos[i].Close
			tempTotalMny := tempHoldMny + holdInfos.LeftMny
			deltaPct := (tempTotalMny - lastDetail.TotalMny) / lastDetail.TotalMny
			if deltaPct < LongBackPct {
				backDetail := simulation.OperationDetail{}
				backDetail.TsCode = tsCode
				backDetail.OpeClose = baseInfos[i].Close
				backDetail.OpeNum = 0
				backDetail.TradeDate = baseInfos[i].TradeDate
				backDetail.HoldNum = holdInfos.HoldNum
				backDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
				backDetail.LeftMny = holdInfos.LeftMny
				backDetail.TotalMny = tempTotalMny
				backDetail.OpeFlag = simulation.NothingOpe
				backDetail.AddDetailToExcelData(&excelData)
			}

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

			// ---------------------------- 下面是卖出操作 ----------------------------------------------------------------
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

				// 如果到达了指定天数并且盈利百分比还是没有到达规定的程度，那么我们卖出
				tempWinPct := (baseInfos[i].Close - timeIndexBuyInfo[0].OpeClose) / timeIndexBuyInfo[0].OpeClose
				if len(timeIndexBuyInfo) > 0 && (i-timeIndexBuyInfo[0].TradeIndex) > NoWinDays && tempWinPct <= TempWinPCt && info.OpeFlag != simulation.HoldFlag {
					tempOpeNum += timeIndexBuyInfo[0].OpeNum
					timeIndexBuyInfo[0].HasSold = true
					timeIndexBuyInfo = timeIndexBuyInfo[1:]
				} else {
					break
				}
			}

			// 检查下是否有达到盈利标准的股票
			for k := 0; k < len(buyPriceOrderInfo); k++ {
				// 已经卖出的去掉
				if buyPriceOrderInfo[k].HasSold {
					buyPriceOrderInfo = append(buyPriceOrderInfo[:k], buyPriceOrderInfo[k+1:]...)
					k--
					continue
				}

				// 判定盈利百分比
				tempWinPct := (baseInfos[i].Close - buyPriceOrderInfo[k].OpeClose) / buyPriceOrderInfo[k].OpeClose
				// 如果是失利了，那么直接卖出，去TMD的
				if tempWinPct < 0 && math.Abs(tempWinPct) >= NoWinPct {
					tempOpeNum += buyPriceOrderInfo[k].OpeNum
					buyPriceOrderInfo[k].HasSold = true
					buyPriceOrderInfo = append(buyPriceOrderInfo[:k], buyPriceOrderInfo[k+1:]...)
					k--
					continue
				} else {
					// 如果是相比较于最大盈利百分比，达到了最大回撤的时候，选择卖出
					maxBackPct := buyPriceOrderInfo[k].MaxWinPct * BackSoldPct
					if tempWinPct <= buyPriceOrderInfo[k].MaxWinPct-maxBackPct && buyPriceOrderInfo[k].TradeIndex != i {
						tempOpeNum += buyPriceOrderInfo[k].OpeNum
						buyPriceOrderInfo[k].HasSold = true
						buyPriceOrderInfo = append(buyPriceOrderInfo[:k], buyPriceOrderInfo[k+1:]...)
						k--
						continue
					}
					// 更新一下最大盈利百分比
					if tempWinPct > buyPriceOrderInfo[k].MaxWinPct {
						buyPriceOrderInfo[k].MaxWinPct = tempWinPct
					}
				}

			}
			// 卖出都是按手为单位(100的整数)
			realOpeNum := int(tempOpeNum) - (int(tempOpeNum) % 100)
			if realOpeNum == 0 {
				addDeltaInfo(&lastDetail, &baseInfos[i], &excelData)
				// 统计信息计算
				updateStatisticInfo(&lastDetail, &baseInfos[i], &statisticInfo)
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

			// 统计信息计算
			updateStatisticInfo(&lastDetail, &baseInfos[i], &statisticInfo)
		}
		if lastDetail.TotalMny > InitMny {
			simulateRst.WinNum += 1
			winPct := (lastDetail.TotalMny - InitMny) / InitMny
			if winPct > simulateRst.MaxWinPct {
				simulateRst.MaxWinPct = winPct
			}
			//simulateRst.WinStockCodes += baseInfos[0].TsCode
		} else if lastDetail.TotalMny < InitMny {
			simulateRst.LostNum += 1
			winPct := (InitMny - lastDetail.TotalMny) / InitMny
			if winPct > simulateRst.MaxLostPct {
				simulateRst.MaxLostPct = winPct
			}
			//simulateRst.LostStockCodes += baseInfos[0].TsCode
		}
		// 最后一次统计信息赋值
		lastTotalMny := float64(lastDetail.HoldNum)*baseInfos[len(baseInfos)-1].Close + lastDetail.LeftMny
		lastWinPct := lastTotalMny / InitMny
		statisticInfo.FinalTotalMny = lastTotalMny
		statisticInfo.FinalWinPct = lastWinPct
		simulateRst.DetailInfo = append(simulateRst.DetailInfo, statisticInfo)

		// 将实时分析结果写入到Excel文件当中去
		fileName := tsCode + ".xlsx"
		excelWriter := file.New(fileName, dirName)
		excelWriter.Write(excelData)
	}
	channel <- simulateRst
}

func addDeltaInfo(lastDetail *simulation.OperationDetail, baseInfo *results.StockBaseInfo, excelData *file.ExcelData) {
	detail := simulation.OperationDetail{
		TsCode:     lastDetail.TsCode,
		TsName:     lastDetail.TsName,
		OpeNum:     0,
		OpeClose:   baseInfo.Close,
		TradeDate:  baseInfo.TradeDate,
		HoldNum:    lastDetail.HoldNum,
		HoldMny:    0, // 下面做计算
		LeftMny:    lastDetail.LeftMny,
		OpeFlag:    simulation.NothingOpe,
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

func updateStatisticInfo(lastDetail *simulation.OperationDetail, baseInfo *results.StockBaseInfo, statisticInfo *simulation.SingleStockSimulateRst) {
	// 计算下当日的总金额
	totalMny := float64(lastDetail.HoldNum)*baseInfo.Close + lastDetail.LeftMny
	winPct := totalMny / InitMny
	if totalMny > statisticInfo.HighestMny {
		statisticInfo.HighestMny = totalMny
		statisticInfo.HighestPct = winPct
		statisticInfo.HighestDay = baseInfo.TradeDate
	} else if statisticInfo.LowestMny > totalMny {
		statisticInfo.LowestMny = totalMny
		statisticInfo.LowestPct = winPct
		statisticInfo.LowestDay = baseInfo.TradeDate
	}
}
