package trackSimulate

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
	DefaultThreadNum = 100
	// 原先的默认值是15，现在改成5试下效果，对于LonggSimulate
	NoWinDays    = 5 // 到达指定天数仍然没有盈利，就卖出操作（全仓）
	TargetWinPct = 0.15
	NoWinPct     = 0.03  // 每笔亏损达到多少的时候就会卖出，不管是否到了NoWinDays指定的天数（TODO）
	LongBackPct  = -0.08 // 最大回撤达到这个百分比的时候，就在最终的输出结果当中做一个反馈

	DisplayDeltaPct = 0.05 // 当当前持仓金额相比于上次持仓金额的百分比达到该数值时，即便无操作也将该条结果显示到最终的交易明细当中
)

var mutex sync.Mutex

// TODO -- 看下盈利比率的中位数是什么？？？
// 跟踪每一笔交易，决定适当时机做买入卖出操作
// 1. 当某次买入的票子持有达到NoWinDays天的时候，不管结果如何，都做卖出操作（或者换种策略，持有就持有，当稍微有盈利了之后才卖出呢？）
// 2. 当某次买入的票子盈利达到了TargetWinPct的时候，我们就做卖出操作（此处可优化，比如让利润奔跑）
// 3. 当多次操作持续性失利的时候，我们可以考虑尽量卖出操作
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
		go singleSimulate(&currIndex, stockList, channel, &waitGroup, dirName)
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

func singleSimulate(index *int, stockList []string, channel chan SimulateRst, waitGroup *sync.WaitGroup, dirName string) {
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

		// 记录买入信息
		var timeIndexBuyInfo []*OperationDetail
		var buyPriceOrderInfo []*OperationDetail

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
		//retOpeTime := simulation.LongEmaSimulate(baseInfos)
		// 开始做分析
		//lostCount := 0		// 失利次数
		var lastDetail OperationDetail
		lastDetail.TotalMny = InitMny
		for i, info := range retOpeTime {
			// 每次开始之前需要检查一下是不是达到了最大回撤的百分比
			tempHoldMny := float64(holdInfos.HoldNum) * baseInfos[i].Close
			tempTotalMny := tempHoldMny + holdInfos.LeftMny
			deltaPct := (tempTotalMny - lastDetail.TotalMny) / lastDetail.TotalMny
			if deltaPct < LongBackPct {
				backDetail := OperationDetail{}
				backDetail.TsCode = tsCode
				backDetail.OpeClose = baseInfos[i].Close
				backDetail.OpeNum = 0
				backDetail.TradeDate = baseInfos[i].TradeDate
				backDetail.HoldNum = holdInfos.HoldNum
				backDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
				backDetail.LeftMny = holdInfos.LeftMny
				backDetail.TotalMny = tempTotalMny
				backDetail.OpeFlag = NothingOpe
				backDetail.AddDetailToExcelData(&excelData)
			}

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
				// FIXME -- 此处指定了买入的必须盈利了才能卖出 -- addDeltaInfo(&lastDetail, &baseInfos[i], &excelData)
				// Important -- 此处加个判定-如果是今天相比较于昨天还是上涨的话，那么我们就不卖出了
				// TODO -- 此处改动是有效地，那么就可以另外考虑一下上涨过程当中的部分回撤，OK？
				if len(timeIndexBuyInfo) > 0 && (i-timeIndexBuyInfo[0].TradeIndex) > NoWinDays && baseInfos[i].Close < baseInfos[i-1].Close {
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
				// 当模拟程序告诉你说要卖出的时候才卖出呢？
				// FIXME -- 此处指定了只有当模拟程序发出卖出信号的时候，才能能够卖出（使盈利最大化）
				if tempWinPct > TargetWinPct && info.OpeFlag == SoldFlag {
					tempOpeNum += buyPriceOrderInfo[k].OpeNum
					buyPriceOrderInfo[k].HasSold = true
					buyPriceOrderInfo = append(buyPriceOrderInfo[:k], buyPriceOrderInfo[k+1:]...)
					k--
				}
			}
			// 卖出都是按手为单位(100的整数)
			realOpeNum := int(tempOpeNum) - (int(tempOpeNum) % 100)
			if realOpeNum == 0 {
				addDeltaInfo(&lastDetail, &baseInfos[i], &excelData)
				continue
			}
			// 更新持仓信息
			holdInfos.HoldNum -= realOpeNum
			holdInfos.LeftMny += float64(realOpeNum) * baseInfos[i].Close

			// 更新操作信息
			soldDetail := OperationDetail{}
			soldDetail.TsCode = tsCode
			soldDetail.OpeClose = baseInfos[i].Close
			soldDetail.OpeNum = realOpeNum
			soldDetail.TradeDate = baseInfos[i].TradeDate
			soldDetail.HoldNum = holdInfos.HoldNum
			soldDetail.HoldMny = float64(holdInfos.HoldNum) * baseInfos[i].Close
			soldDetail.LeftMny = holdInfos.LeftMny
			soldDetail.TotalMny = soldDetail.HoldMny + soldDetail.LeftMny
			soldDetail.OpeFlag = SoldDisplay
			soldDetail.AddDetailToExcelData(&excelData)
			lastDetail = soldDetail
		}
		if lastDetail.TotalMny > InitMny {
			simulateRst.winNum += 1
			winPct := (lastDetail.TotalMny - InitMny) / InitMny
			if winPct > simulateRst.maxWinPct {
				simulateRst.maxWinPct = winPct
			}
			simulateRst.WinStockCodes += baseInfos[0].TsCode
		} else {
			simulateRst.lostNum += 1
			winPct := (InitMny - lastDetail.TotalMny) / InitMny
			if winPct > simulateRst.maxLostPct {
				simulateRst.maxLostPct = winPct
			}
			simulateRst.LostStockCodes += baseInfos[0].TsCode
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
