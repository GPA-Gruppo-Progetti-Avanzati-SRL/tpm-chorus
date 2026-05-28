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

	// Costanti per AmtFmtConv
	NumFmtDecimal = "decimal" // valore con separatore decimale esplicito (es. "7.10", "7,10", "-0.50")
	NumFmtInt2    = "int2"    // intero, 2 decimali impliciti:  710      rappresenta   7.10
	NumFmtInt3    = "int3"    // intero, 3 decimali impliciti:  7100     rappresenta   7.100
	NumFmtInt4    = "int4"    // intero, 4 decimali impliciti:  71000    rappresenta   7.1000
	NumFmtInt5    = "int5"    // intero, 5 decimali impliciti:  710000   rappresenta   7.10000
	NumFmtInt6    = "int6"    // intero, 6 decimali impliciti:  7100000  rappresenta   7.100000

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
