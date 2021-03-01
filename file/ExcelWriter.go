package file

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"os"
	"strconv"
)

func init() {
	DefaultPreWorkspace = "D:\\pystock"
}

var DefaultPreWorkspace string

type ExcelWriter struct {
	FileName string
	dirName  string
	f        *excelize.File
}

func New(fileName string, outDirName string) *ExcelWriter {
	writer := ExcelWriter{
		FileName: fileName,
		dirName:  outDirName,
		f:        excelize.NewFile(),
	}
	return &writer
}

func (excelWriter *ExcelWriter) Write(data ExcelData) {
	f := excelize.NewFile()

	// 按列写就完了
	for i, value := range data.Columns {
		tempColumnName := ConvertIndexToColumn(i)
		columnName := tempColumnName + strconv.Itoa(1)
		// 首先写行表体
		_ = f.SetCellValue("Sheet1", columnName, value)

		// 然后开始循环写表体
		columnData := data.Data[value]
		for j, tempData := range columnData {
			name := tempColumnName + strconv.Itoa(j+2)
			_ = f.SetCellValue("Sheet1", name, tempData)
		}
	}

	checkThenCreateDir(DefaultPreWorkspace + "\\" + excelWriter.dirName)
	finalFilePath := DefaultPreWorkspace + "\\" + excelWriter.dirName + "\\" + excelWriter.FileName
	if err := f.SaveAs(finalFilePath); err != nil {
		fmt.Println(err)
	}
}

func checkThenCreateDir(dirName string) {
	_, err := os.Stat(dirName)
	if err == nil {
		return
	}

	// 创建文件夹
	err = os.Mkdir(dirName, os.ModePerm)
	if err != nil {
		panic("创建文件夹失败！！" + dirName + " " + err.Error())
	}
}

func ConvertIndexToColumn(i int) string {
	retString := ""
	for {
		tempString := string(rune(65 + i%26))
		retString = tempString + retString

		if i <= 25 {
			break
		}
		i = i/26 - 1
	}
	return retString
}
