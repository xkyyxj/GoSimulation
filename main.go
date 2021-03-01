package main

import (
	_ "database/sql"
	"stock_simulate/datacenter"
	"stock_simulate/trackSimulate"
)

var SingleDataCenter *datacenter.DataCenter

func init() {
	SingleDataCenter = &datacenter.DataCenter{}
	SingleDataCenter.Initialize()
}

func main() {
	//println(file.ConvertIndexToColumn(26))
	//
	//// 测试一下批量写入Excel数据的问题
	//excelData := file.ExcelData{
	//	Columns: make([]string, 0),
	//	Data: make(map[string][]interface{}),
	//}
	//columns := []string {"hehe", "dada"}
	//excelData.Columns = columns
	//
	//columns1Data := []int {123, 456}
	//columns2Data := []int {345, 890}
	//excelData.SetData("hehe", columns1Data)
	//excelData.SetData("dada", columns2Data)
	//writer := file.New("first.xlsx", "temp")
	//writer.Write(excelData)
	//tempDetail := trackSimulate.OperationDetail{}
	//tempDetail.HasSold = true
	//var timeIndexBuyInfo []*trackSimulate.OperationDetail
	//var buyPriceOrderInfo []*trackSimulate.OperationDetail
	//timeIndexBuyInfo = append(timeIndexBuyInfo, &tempDetail)
	//buyPriceOrderInfo = append(buyPriceOrderInfo, &tempDetail)
	//timeIndexBuyInfo[0].HasSold = false
	//println("has sold is {}", buyPriceOrderInfo[0].HasSold)
	trackSimulate.Simulate("track_history_down_000001")
	//simulation.Simulate("ema_simulate_005")
	//allSimulate.UpGoSimualte()
}

func testArray(array1 [2]int) {
	array1[0] = 7
}
