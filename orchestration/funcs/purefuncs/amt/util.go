package amt

import "fmt"

var amountConversionsMap = map[string]func(string) string{
	fmt.Sprintf(ConversionMapKetFormat, Cent, Mill):      func(a string) string { return a + "0" },
	fmt.Sprintf(ConversionMapKetFormat, Cent, MicroCent): func(a string) string { return a + "0000" },
	fmt.Sprintf(ConversionMapKetFormat, Cent, Cent):      func(a string) string { return a },
	fmt.Sprintf(ConversionMapKetFormat, Mill, Mill):      func(a string) string { return a },
	fmt.Sprintf(ConversionMapKetFormat, Mill, Cent): func(a string) string {
		if len(a) > 1 {
			return a[0 : len(a)-1]
		}

		return "0"
	},

	fmt.Sprintf(ConversionMapKetFormat, MicroCent, Cent): func(a string) string {
		if len(a) > 4 {
			return a[0 : len(a)-4]
		}

		return "0"
	},
}

func toDecimalFormat(s string) string {
	if len(s) < 3 {
		s = "000" + s
		s = s[len(s)-3:]
	}
	return s[:len(s)-2] + "." + s[len(s)-2:]
}

func fmtConvCent2Cent(i int64, d string) string {
	return fmt.Sprint(i)
}

func fmtConvCent2Mill(i int64, d string) string {
	return fmt.Sprintf("%d0", i)
}

func fmtConvCent2MicroCent(i int64, d string) string {
	return fmt.Sprintf("%d0000", i)
}

func fmtConvCent2Decimal(i int64, d string) string {
	var si string
	sd := "00"
	if i >= 100 {
		s := fmt.Sprint(i)
		sd = s[len(s)-2:]
		si = s[:len(s)-2]
	} else {
		si = "0"
		sd = fmt.Sprintf("%02d", i)
	}

	return fmt.Sprintf("%s.%s", si, sd)
}

func fmtConvMill2Mill(i int64, d string) string {
	return fmt.Sprint(i)
}

func fmtConvMillToCent(i int64, d string) string {
	if i >= 10 {
		s := fmt.Sprint(i)
		return s[0 : len(s)-1]
	}

	return "0"
}

func fmtConvMicroCent2Cent(i int64, d string) string {
	if i >= 10000 {
		s := fmt.Sprint(i)
		return s[0 : len(s)-4]
	}

	return "0"
}

func fmtConvDecimal2Cent(i int64, d string) string {

	switch len(d) {
	case 0:
		if i > 0 {
			d = "00"
		}
	case 1:
		d = d + "0"
	case 2:
	default:
		d = d[0 : len(d)-1]
	}

	return fmt.Sprintf("%d%s", i, d)
}
