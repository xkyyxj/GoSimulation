package simulation

import (
	"stock_simulate/file"
	"strconv"
)

const (
	BuyFlag  = 1
	SoldFlag = 2
	Nothing  = 3

	BuyDisplay  = "买入"
	SoldDisplay = "卖出"
)

type StockHoldInfo struct {
	InitMny float64
	HoldNum int
	LeftMny float64
}

type SimulateRst struct {
	WinNum       int
	LostNum      int
	MaxWinPct    float64
	MaxLostPct   float64 // 此处应该是正数
	MaxWinStock  string
	maxLostStock string
	parameter    string
}

type OperateInfo struct {
	OpeFlag    int
	OpePercent float64
}

type OperationDetail struct {
	TsCode     string
	TsName     string
	OpeNum     int
	OpeClose   float64
	TradeDate  string
	HoldNum    int
	HoldMny    float64
	LeftMny    float64
	TotalMny   float64
	OpeFlag    string
	TradeIndex int
	HasSold    bool
}

func (operationDetail *OperationDetail) AddDetailToExcelData(excelData *file.ExcelData) {
	if excelData.Columns == nil || len(excelData.Columns) == 0 {
		excelData.Columns = []string{"股票编码", "操作数量", "操作类型", "收盘价", "交易日期", "持仓数量", "持仓金额", "剩余金额", "总金额"}
	}
	excelData.Data["股票编码"] = append(excelData.Data["股票编码"], operationDetail.TsCode)
	excelData.Data["操作数量"] = append(excelData.Data["操作数量"], operationDetail.OpeNum)
	excelData.Data["操作类型"] = append(excelData.Data["操作类型"], operationDetail.OpeFlag)
	excelData.Data["收盘价"] = append(excelData.Data["收盘价"], operationDetail.OpeClose)
	excelData.Data["交易日期"] = append(excelData.Data["交易日期"], operationDetail.TradeDate)
	excelData.Data["持仓数量"] = append(excelData.Data["持仓数量"], operationDetail.HoldNum)
	excelData.Data["持仓金额"] = append(excelData.Data["持仓金额"], operationDetail.HoldMny)
	excelData.Data["剩余金额"] = append(excelData.Data["剩余金额"], operationDetail.LeftMny)
	excelData.Data["总金额"] = append(excelData.Data["总金额"], operationDetail.TotalMny)
}

func (simulateRst *SimulateRst) ToString() string {
	outString := ""
	outString += "winNumber is: " + strconv.Itoa(simulateRst.WinNum) + "\r\n"
	outString += "LostNum is: " + strconv.Itoa(simulateRst.LostNum) + "\r\n"
	outString += "MaxWinPct is: " + strconv.FormatFloat(simulateRst.MaxWinPct, 'E', 2, 64) + "\r\n"
	outString += "MaxLostPct is: " + strconv.FormatFloat(simulateRst.MaxLostPct, 'E', 2, 64) + "\r\n"
	outString += "MaxWinStock is " + simulateRst.MaxWinStock + "\r\n"
	outString += "maxLostStock is " + simulateRst.maxLostStock + "\r\n"
	outString += "parameter is " + simulateRst.parameter + "\r\n"
	return outString
}
