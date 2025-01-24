package amt

const (
	Dime          string = "dime"
	MicroCent            = "micro"
	Mill                 = "mill"
	Cent                 = "cent"
	Decimal              = "decimal"
	DecimalCent          = "decimal-2"
	DecimalMillis        = "decimal-3"

	DeciMill = "deci-mill"

	ConversionMapKetFormat = "%s_to_%s"

	AmountOpAdd  = "add"
	AmountOpDiff = "diff"
)

var IntegralAmtFormats = map[string][]struct{}{
	Dime:      {},
	Cent:      {},
	Mill:      {},
	MicroCent: {},
}
