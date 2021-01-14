package allSimulate

import (
	"math/rand"
	"stock_simulate/datacenter"
	"stock_simulate/file"
	"strconv"
	"sync"
	"time"
)

const (
	AllEMAField        = 3
	AllEMAUpBuyCount   = 3
	DefaultPct         = 0.1 // 默认买入的百分比
	AllEMADownSoldDays = 2   // 到达多少天的下降了考虑下卖出
	AllEWADownSoldPct  = -0.03

	InitMny          = 100000
	DefaultThreadNum = 100
)

var opeInfoCount int = 0

type UpStockInfo struct {
	tsCode string
}

type HoldInfo struct {
	tsCode string
	inDate string
	price  float64
	number int
}

type OperateInfo struct {
	TsCode    string
	TradeDate string
	Price     float64
	Number    int
	OpeType   string
	HoldMny   float64
	LeftMny   float64
	TotalMny  float64
}

type MnyInfo struct {
	initMny float64
	leftMny float64
}

func (operationDetail *OperateInfo) appendOpeInfoToExcel(excelData *file.ExcelData) {
	opeInfoCount++
	if opeInfoCount%100 == 0 {
		println(opeInfoCount)
	}
	if excelData.Columns == nil || len(excelData.Columns) == 0 {
		excelData.Columns = []string{"股票编码", "操作数量", "操作类型", "收盘价", "交易日期", "持仓数量", "持仓金额", "剩余金额", "总金额"}
	}
	excelData.Data["股票编码"] = append(excelData.Data["股票编码"], operationDetail.TsCode)
	excelData.Data["操作数量"] = append(excelData.Data["操作数量"], operationDetail.Number)
	excelData.Data["操作类型"] = append(excelData.Data["操作类型"], operationDetail.OpeType)
	excelData.Data["收盘价"] = append(excelData.Data["收盘价"], operationDetail.Price)
	excelData.Data["交易日期"] = append(excelData.Data["交易日期"], operationDetail.TradeDate)
	excelData.Data["持仓金额"] = append(excelData.Data["持仓金额"], operationDetail.HoldMny)
	excelData.Data["剩余金额"] = append(excelData.Data["剩余金额"], operationDetail.LeftMny)
	excelData.Data["总金额"] = append(excelData.Data["总金额"], operationDetail.TotalMny)
}

// 所谓的追涨杀跌，既然跌的一直跌，那么就追涨杀跌吧，有好处就赚了
func UpGoSimualte() {
	dataCenter := datacenter.GetInstance()
	stockList := dataCenter.QueryStockCodes("")
	var waitGroup sync.WaitGroup
	// 根据基础指数来
	dataStr := make([]string, 1000)
	indiSql := "select trade_date from stock_index_baseinfo where ts_code='000001.SH' order by trade_date desc limit 1000"
	_ = dataCenter.Db.Select(&dataStr, indiSql)

	// 准备初始数据，按照线程数分组
	eachGrpNum := len(stockList) / DefaultThreadNum
	eachGrpNum = eachGrpNum + 1
	stockGrp := [DefaultThreadNum][]string{}
	tempSlice := make([]string, eachGrpNum)
	for i := 0; i < DefaultThreadNum; i++ {
		for j := 0; j < eachGrpNum; j++ {
			if i*eachGrpNum+j < len(stockList) {
				tempSlice[j] = stockList[i*eachGrpNum+j]
			}
		}
		stockGrp[i] = tempSlice
		tempSlice = make([]string, eachGrpNum)
	}

	// 准备好业务初始化数据
	holdInfos := make([]HoldInfo, 100)
	mnyInfo := MnyInfo{
		initMny: InitMny,
		leftMny: InitMny,
	}

	// 记录所有的操作信息
	excelData := file.ExcelData{}

	// 开始循环操作
	channelSlice := make([]<-chan []string, DefaultThreadNum)
	for _, date := range dataStr {
		waitGroup.Add(DefaultThreadNum)
		for j := 0; j < DefaultThreadNum; j++ {
			channel := make(chan []string, 300)
			go findUpStock(stockGrp[j], date, channel, &waitGroup)
			channelSlice[j] = channel
		}
		waitGroup.Wait()

		println("find stock finished!!")

		allUpStock := make([]string, 400)
		for j := 0; j < DefaultThreadNum; j++ {
			tempValue := <-channelSlice[j]
			allUpStock = append(allUpStock, tempValue...)
		}

		// 查看当前持有股票是否还在上涨过程当中（allUpStock包含该只股票）
		// 如果不包含了，就卖出操作
		newHoldInfos := make([]HoldInfo, 400)
		for _, hold := range holdInfos {
			stillUp := false
			for _, str := range allUpStock {
				if hold.tsCode == str {
					stillUp = true
					newHoldInfos = append(newHoldInfos, hold)
				}
			}

			if !stillUp {
				// TODO -- 现在的卖出逻辑是全部卖出
				soldMny := soldStock(hold.tsCode, date, hold.number, &mnyInfo, holdInfos, &excelData)
				mnyInfo.leftMny = mnyInfo.leftMny + soldMny
			}
		}
		holdInfos = newHoldInfos

		if len(allUpStock) == 0 {
			continue
		}
		rand.Seed(time.Now().UnixNano())
		stockIndex := rand.Intn(len(allUpStock))
		targetStr := allUpStock[stockIndex]
		// 默认买入的百分比
		retHoldInfo := buyStock(targetStr, date, DefaultPct, mnyInfo)
		holdInfos = append(holdInfos, retHoldInfo)
	}

	// 记录操作信息
	lastDate := dataStr[len(dataStr)-1]
	lastHoldMny := currHoldMnyInfo(holdInfos, lastDate)
	opeInfo := OperateInfo{
		TsCode:    "",
		TradeDate: lastDate,
		Price:     0,
		Number:    0,
		OpeType:   "卖出",
		HoldMny:   lastHoldMny,
		LeftMny:   mnyInfo.leftMny,
		TotalMny:  lastHoldMny + mnyInfo.leftMny,
	}
	opeInfo.appendOpeInfoToExcel(&excelData)
	excelWriter := file.New("final.xlsx", "allSimulate")
	excelWriter.Write(excelData)
}

// 查找连续上涨的股票，通过ema_value来判定
func findUpStock(stockList []string, dataStr string, channel chan []string, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	selectedStock := make([]string, DefaultThreadNum)
	emaField := "ema_" + strconv.Itoa(AllEMAField)
	emaLimitNum := strconv.Itoa(AllEMAUpBuyCount)
	dataCenter := datacenter.GetInstance()
	for _, info := range stockList {
		targetEmaValue := make([]float64, AllEMAUpBuyCount)
		querySql := "select " + emaField + " from ema_value where ts_code='" + info + "' and trade_date < '" + dataStr + "' order by trade_date desc limit "
		querySql += emaLimitNum
		_ = dataCenter.Db.Select(&targetEmaValue, querySql)
		alwaysUp := true
		// 没有数据的话，就直接跳到下一只股票
		if targetEmaValue == nil || len(targetEmaValue) == 0 {
			continue
		}
		preEmaValue := targetEmaValue[0]
		for _, emaValue := range targetEmaValue {
			alwaysUp = alwaysUp && emaValue > preEmaValue
		}

		if alwaysUp {
			selectedStock = append(selectedStock, info)
		}
	}
	channel <- selectedStock
}

// 卖出股票，返回卖出得到的钱
func soldStock(tsCode string, dateStr string, soldNum int, mnyInfo *MnyInfo, currHold []HoldInfo, excelData *file.ExcelData) float64 {
	// 第一步查询对应的价格数据
	closeVal := make([]float64, 1)
	qrySql := "select close from stock_base_info where ts_code='" + tsCode + "' and trade_date='" + dateStr + "'"
	dbInstance := datacenter.GetInstance()
	_ = dbInstance.Db.Select(&closeVal, qrySql)

	holdMny := currHoldMnyInfo(currHold, dateStr)

	// 记录操作信息
	opeInfo := OperateInfo{
		TsCode:    tsCode,
		TradeDate: dateStr,
		Price:     0,
		Number:    soldNum,
		OpeType:   "卖出",
		HoldMny:   holdMny,
		LeftMny:   mnyInfo.leftMny,
		TotalMny:  holdMny + mnyInfo.leftMny,
	}
	opeInfo.appendOpeInfoToExcel(excelData)

	return float64(soldNum) * closeVal[0]
}

// 买入股票，返回买入信息
func buyStock(tsCode string, dateStr string, buyPct float64, mnyInfo MnyInfo) HoldInfo {
	holdInfo := HoldInfo{
		tsCode: tsCode,
		inDate: dateStr,
	}

	closeVal := make([]float64, 1)
	qrySql := "select close from stock_base_info where ts_code='" + tsCode + "' and trade_date='" + dateStr + "'"
	dbInstance := datacenter.GetInstance()
	_ = dbInstance.Db.Select(&closeVal, qrySql)

	holdInfo.price = closeVal[0]

	tempBuyMny := mnyInfo.leftMny * buyPct
	if tempBuyMny > mnyInfo.leftMny {
		tempBuyMny = mnyInfo.leftMny
	}
	// 买入都是按手为单位(100的整数)
	tempBuyNum := tempBuyMny / closeVal[0]
	intBuyNum := int(tempBuyNum)
	intBuyNum = intBuyNum - (intBuyNum % 100)

	realBuyMny := float64(intBuyNum) * closeVal[0]
	mnyInfo.leftMny = mnyInfo.leftMny - realBuyMny
	holdInfo.number = intBuyNum

	return holdInfo
}

// 统计当前的金额信息
func currHoldMnyInfo(infos []HoldInfo, date string) float64 {
	dataCenter := datacenter.GetInstance()
	var holdMny float64 = 0
	for _, info := range infos {
		closeVal := make([]float64, 1)
		querySql := "select close from stock_base_info where ts_code='" + info.tsCode + "' and trade_date='" + date + "'"
		_ = dataCenter.Db.Select(&closeVal, querySql)
		holdMny += closeVal[0] * float64(info.number)
	}
	return holdMny
}
