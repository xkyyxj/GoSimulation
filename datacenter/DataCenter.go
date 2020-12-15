package datacenter

import (
	_ "database/sql"
	_ "github.com/go-sql-driver/mysql"
	"stock_simulate/results"
	"sync"

	"github.com/jmoiron/sqlx"
)

var instance *DataCenter
var once sync.Once

func GetInstance() *DataCenter {
	once.Do(func(){
		instance=&DataCenter{}
		instance.Initialize()
	})
	return instance
}

type DataCenter struct {
	Db *sqlx.DB
}

func (dataCenter *DataCenter) Initialize() {
	// 此处默认写死了
	db, err := sqlx.Open("mysql","root:123@tcp(localhost:3306)/stock?charset=utf8")
	if err != nil {
		panic("数据库连接失败，请检查数据库！")
	}
	dataCenter.Db = db
}

// 通用查询之一：查询所有的股票列表
// 只返回股票的编码列表
func (dataCenter *DataCenter) QueryStockCodes(wherePart string) []string {
	var stockCodes []string
	sql := "select ts_code from stock_list where "
	if len(wherePart) == 0 {
		wherePart = " market in ('主板','中小板')"
	}
	sql = sql + wherePart
	err := dataCenter.Db.Select(&stockCodes, sql)
	if err != nil {
		panic(err)
	}
	return stockCodes
}

// 通用查询之一：查询所有的基本信息
func (dataCenter *DataCenter) QueryStockBaseInfo(wherePart string) []results.StockBaseInfo {
	var infos []results.StockBaseInfo
	sql := "select * from stock_base_info where "
	if len(wherePart) != 0 {
		sql = sql + wherePart
	}
	_ = dataCenter.Db.Select(&infos, sql)
	return infos
}