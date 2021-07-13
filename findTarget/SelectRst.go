package findTarget

import "stock_simulate/file"

type SingleSelectRst struct {
	TsCode     string
	TsName     string
	CurrClose  float64
	TwoDayPct  float64
	FiveDayPct float64
	SourceFun  string
}

type SelectRst struct {
	SelectTime string
	AllRst     []SingleSelectRst
}

func (selectRst *SelectRst) ConvertSimulateRstToExcelData(excelData *file.ExcelData) {
	if excelData.Columns == nil || len(excelData.Columns) == 0 {
		excelData.Columns = []string{"股票编码", "股票名称", "当前收盘价", "两日上涨幅度", "五日上涨幅度", "来源算法"}
	}
	for _, item := range selectRst.AllRst {
		excelData.Data["股票编码"] = append(excelData.Data["股票编码"], item.TsCode)
		excelData.Data["股票名称"] = append(excelData.Data["股票名称"], item.TsName)
		excelData.Data["当前收盘价"] = append(excelData.Data["当前收盘价"], item.CurrClose)
		excelData.Data["两日上涨幅度"] = append(excelData.Data["两日上涨幅度"], item.TwoDayPct)
		excelData.Data["五日上涨幅度"] = append(excelData.Data["五日上涨幅度"], item.FiveDayPct)
		excelData.Data["来源算法"] = append(excelData.Data["来源算法"], item.SourceFun)
	}
}

func (selectRst *SelectRst) MergeRst(input *SelectRst) {
	for _, innerItem := range input.AllRst {
		hasItem := false
		for _, item := range selectRst.AllRst {
			if item.TsCode == innerItem.TsCode {
				hasItem = true
				break
			}
		}

		if !hasItem {
			selectRst.AllRst = append(selectRst.AllRst, innerItem)
		}
	}
}
