package results

type StockBaseInfo struct {
	TsCode     string  `db:"ts_code"`
	Open       float64 `db:"open"`
	Close      float64 `db:"close"`
	High       float64 `db:"high"`
	Low        float64 `db:"low"`
	Vol        float64 `db:"vol"`
	Amount     float64 `db:"amount"`
	PreClose   float64 `db:"pre_close"`
	Change     float64 `db:"change"`
	PctChg     float64 `db:"pct_chg"`
	TradeDate  string  `db:"trade_date"`
	AfterOpen  float64
	AfterClose float64
	AfterLow   float64
	AfterHigh  float64
}
