package simulation

import (
	"stock_simulate/file"
	"strconv"
)

const (
	BuyFlag  = 1
	SoldFlag = 2
	Nothing  = 3
	HoldFlag = 4 // 推荐持有，仅供推荐

	BuyDisplay  = "买入"
	SoldDisplay = "卖出"
	NothingOpe  = "无操作"
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
	MaxLostStock string
	Parameter    string
	DetailInfo   []SingleStockSimulateRst
}

/**
单只股票的模拟结果
*/
type SingleStockSimulateRst struct {
	TsCode        string
	TsName        string
	LowestMny     float64 // 最低金额
	LowestPct     float64 // 最低盈利百分比（可能为负值）
	LowestDay     string  // 最低金额日期
	HighestMny    float64 // 最高金额
	HighestPct    float64 // 最高盈利百分比
	HighestDay    string  // 最高金额日期
	FinalTotalMny float64 // 最终金额
	FinalWinPct   float64 // 最终盈利百分比
	CurrTotalMny  float64 // 当前的总金额（为了统计方便加的字段 -- 废弃，在想啥？）
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
	MaxWinPct  float64
}

func (simulateRst *SimulateRst) ConvertSimulateRstToExcelData(excelData *file.ExcelData) {
	if excelData.Columns == nil || len(excelData.Columns) == 0 {
		excelData.Columns = []string{"股票编码", "股票名称", "最低盈利百分比", "最低金额", "最低金额日期", "最高盈利百分比", "最高金额", "最高金额日期", "最终盈利百分比", "最终金额"}
	}
	for _, item := range simulateRst.DetailInfo {
		excelData.Data["股票编码"] = append(excelData.Data["股票编码"], item.TsCode)
		excelData.Data["股票名称"] = append(excelData.Data["股票名称"], item.TsName)
		excelData.Data["最终盈利百分比"] = append(excelData.Data["最终盈利百分比"], item.FinalWinPct)
		excelData.Data["最低盈利百分比"] = append(excelData.Data["最低盈利百分比"], item.LowestPct)
		excelData.Data["最低金额"] = append(excelData.Data["最低金额"], item.LowestMny)
		excelData.Data["最低金额日期"] = append(excelData.Data["最低金额日期"], item.LowestDay)
		excelData.Data["最终金额"] = append(excelData.Data["最终金额"], item.FinalTotalMny)
		excelData.Data["最高盈利百分比"] = append(excelData.Data["最高盈利百分比"], item.HighestPct)
		excelData.Data["最高金额"] = append(excelData.Data["最高金额"], item.HighestMny)
		excelData.Data["最高金额日期"] = append(excelData.Data["最高金额日期"], item.HighestDay)
	}
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
	outString += "MaxLostStock is " + simulateRst.MaxLostStock + "\r\n"
	outString += "Parameter is " + simulateRst.Parameter + "\r\n"
	return outString
}
