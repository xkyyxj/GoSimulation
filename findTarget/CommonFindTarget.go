package findTarget

import (
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"sync"
)

const (
	DefaultThreadNum = 100
)

var mutex sync.Mutex

func FindTarget(dirName string) {
	dataCenter := datacenter.GetInstance()
	var currIndex int
	currIndex = 0
	stockList := dataCenter.QueryStockCodes("")
	channelSlice := make([]<-chan SelectRst, DefaultThreadNum)
	var waitGroup sync.WaitGroup
	waitGroup.Add(DefaultThreadNum)
	for i := 0; i < DefaultThreadNum; i++ {
		channel := make(chan SelectRst, 10)
		go singleSelect(&currIndex, stockList, channel, &waitGroup, dirName)
		channelSlice[i] = channel
	}

	// 最终结果统计
	waitGroup.Wait()
	println("in here!!")
	finalRst := SelectRst{}
	for _, channel := range channelSlice {
		tempVal := <-channel
		finalRst.MergeRst(&tempVal)
	}

	excelData := file.ExcelData{
		Data: make(map[string][]interface{}),
	}
	finalRst.ConvertSimulateRstToExcelData(&excelData)
	fileName := "selectRst.xlsx"
	excelWriter := file.New(fileName, dirName)
	excelWriter.Write(excelData)
	println("Calculate finished!!!")
}

func singleSelect(index *int, stockList []string, channel chan SelectRst, waitGroup *sync.WaitGroup, dirName string) {
	defer waitGroup.Done()
	// 最终返回结果
	selectRst := SelectRst{}
	for {
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
		baseInfos := dataCenter.QueryStockBaseInfo(" ts_code='" + tsCode + "' order by trade_date desc limit 100")
		if baseInfos == nil || len(baseInfos) == 0 {
			continue
		}

		// 判定是否符合条件
		if judgeIsUpSignal(baseInfos) {
			// 计算相关的上涨幅度
			tempSingleRst := SingleSelectRst{}
			tempSingleRst.TsCode = baseInfos[0].TsCode
			tempSingleRst.CurrClose = baseInfos[0].Close
			tempSingleRst.SourceFun = "两日上涨"
			if len(baseInfos) > 2 {
				tempSingleRst.TwoDayPct = (baseInfos[0].Close - baseInfos[2].Close) / baseInfos[2].Close
			}

			if len(baseInfos) > 5 {
				tempSingleRst.FiveDayPct = (baseInfos[0].Close - baseInfos[5].Close) / baseInfos[5].Close
			}
			selectRst.AllRst = append(selectRst.AllRst, tempSingleRst)
			print("Ok")
		}
	}
	channel <- selectRst
}
