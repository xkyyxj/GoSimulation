package results

type AdjFactorStruct struct {
	TsCode    string  `db:"ts_code"`
	TradeDate string  `db:"trade_date"`
	AdjFactor float64 `db:"adj_factor"`
}
