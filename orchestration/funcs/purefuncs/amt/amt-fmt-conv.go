package amt

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// AmtFmtConv converte un valore numerico da un formato sorgente (srcFormat) a un formato
// destinazione (dstFormat), restituendo sempre una stringa e un eventuale errore.
//
// ── Parametri ───────────────────────────────────────────────────────────────────
//
//   - value     il valore da convertire; può essere int, int64, float64 o string.
//
//   - srcFormat il formato in cui interpretare value. Valori ammessi:
//     • NumFmtDecimal ("decimal") – numero con separatore decimale esplicito,
//     sia "." che "," sono accettati come separatore in input.
//     Es.: "7.10", "7,10", "-0.50", "1500.00"
//     • NumFmtInt2..NumFmtInt6 ("int2".."int6") – numero intero con N cifre
//     decimali implicite. Il valore 710 in int2 rappresenta 7,10.
//     Es.: int2→ 710=7.10 | int3→ 7100=7.100 | int6→ 7000000=7.000000
//
//   - dstFormat il formato dell'output. Stessi valori ammessi di srcFormat.
//     Quando è NumFmtDecimal il numero di cifre decimali nell'output è
//     controllato dal parametro decimals.
//
//   - decimals  numero di cifre decimali nell'output; rilevante solo quando
//     dstFormat == NumFmtDecimal. Ignorato per i formati intN.
//     • -1 → "precisione naturale": vengono mantenute tutte le cifre decimali
//     presenti nel valore sorgente dopo la conversione di scala, eliminando
//     gli zeri finali non significativi. Se il risultato è intero, il
//     separatore decimale viene omesso.
//     • 0..12 → numero fisso di decimali (valori > 12 vengono limitati a 12).
//     • valori < -1 vengono trattati come -1.
//
//   - decSep    separatore decimale dell'output; rilevante solo quando
//     dstFormat == NumFmtDecimal. Valori ammessi: "" o "." → punto,
//     "," → virgola. Qualsiasi altro valore restituisce errore.
//
// ── Comportamento ───────────────────────────────────────────────────────────────
//
// Internamente il valore viene ridotto alla tripla canonica (neg bool, digits string,
// scale int), dove digits è la stringa di cifre che rappresenta |value × 10^scale|
// senza zeri iniziali. La conversione tra formati si riduce ad aggiungere o
// rimuovere cifre a destra della stringa (append/truncate), senza alcuna
// operazione aritmetica intera: overflow impossibile per qualsiasi dimensione
// dell'input.
//
// Quando si riduce il numero di decimali (es. int3 → int2) si applica il
// troncamento verso zero, non l'arrotondamento.
// Es.: 7.155 (int3=7155) → int2 = 715 (7.15), non 716.
//
// ── Esempi ──────────────────────────────────────────────────────────────────────
//
//	// da intero implicito a intero implicito
	//	AmtFmtConv("710",  "int2", "int3", 0, "")   → "7100",  nil   // 7.10 → 7.100
	//	AmtFmtConv("7100", "int3", "int2", 0, "")   → "710",   nil   // 7.100 → 7.10
	//	AmtFmtConv("7155", "int3", "int2", 0, "")   → "715",   nil   // troncamento
	//
	//	// da decimale a intero implicito
	//	AmtFmtConv("7.10", "decimal", "int3", 0, "")  → "7100", nil
	//	AmtFmtConv("-0.50","decimal", "int2", 0, "")  → "-50",  nil
	//
	//	// da intero implicito a decimale
	//	AmtFmtConv("710",    "int2", "decimal", 2, ".")  → "7.10",    nil
	//	AmtFmtConv("710",    "int2", "decimal", 2, ",")  → "7,10",    nil
	//	AmtFmtConv("-150000","int2", "decimal", 2, ".")  → "-1500.00",nil
	//
	//	// da decimale a decimale (normalizzazione separatore / cifre)
	//	AmtFmtConv("7,10", "decimal", "decimal", 4, ".")  → "7.1000", nil
	//	AmtFmtConv("1500", "decimal", "decimal", 2, ".")  → "1500.00",nil
	//
	//	// decimals=-1 → precisione naturale senza zeri finali
	//	AmtFmtConv("12350", "int3", "decimal", -1, ",")  → "12,35",  nil  // 12.350 → 12.35
	//	AmtFmtConv("12300", "int3", "decimal", -1, ",")  → "12,3",   nil  // 12.300 → 12.3
	//	AmtFmtConv("12000", "int3", "decimal", -1, ",")  → "12",     nil  // 12.000 → 12 (intero)
	//	AmtFmtConv("7,10",  "decimal","decimal",-1, ".")  → "7.1",   nil  // zeri rimossi
	//
	//	// tipi numerici nativi come input
	//	AmtFmtConv(710,         "int2", "int3",    0, "")  → "7100", nil
	//	AmtFmtConv(float64(7.1),"decimal","int3",  0, "")  → "7100", nil
	//
	//	// errori
	//	AmtFmtConv("abc",  "int2",    "int3",    0, "")   → "", error  // non numerico
	//	AmtFmtConv("7.10", "int2",    "int3",    0, "")   → "", error  // sep. in formato int
	//	AmtFmtConv("710",  "int2",    "decimal", 2, ";")  → "", error  // decSep non valido
	//	AmtFmtConv("",     "int2",    "int3",    0, "")   → "", error  // stringa vuota
// amtFmtConvParseDecimals converte il parametro decimals (che può arrivare come int,
// int64 o float64 dall'evaluator delle espressioni `:func(...)`) in un int.
func amtFmtConvParseDecimals(raw interface{}) (int, error) {
	switch d := raw.(type) {
	case int:
		return d, nil
	case int64:
		return int(d), nil
	case float64:
		return int(d), nil
	default:
		return 0, fmt.Errorf("amtFmtConv: parametro decimals non valido (tipo %T)", raw)
	}
}

func AmtFmtConv(value interface{}, srcFormat, dstFormat string, decimals interface{}, decSep string) (string, error) {
	// 0. conversione del parametro decimals (int / int64 / float64 → int)
	dec, err := amtFmtConvParseDecimals(decimals)
	if err != nil {
		return "", err
	}

	// 1. validazione separatore decimale di output
	outSep := byte('.')
	switch decSep {
	case "", ".":
		outSep = '.'
	case ",":
		outSep = ','
	default:
		return "", fmt.Errorf("amtFmtConv: separatore decimale non valido %q (validi: \".\" oppure \",\")", decSep)
	}

	// 2. parsing del valore nella rappresentazione canonica
	neg, digits, scale, err := numConvParse(value, srcFormat)
	if err != nil {
		return "", fmt.Errorf("amtFmtConv: errore nella lettura del valore: %w", err)
	}

	// 3. formattazione nella destinazione richiesta
	result, err := numConvFormat(neg, digits, scale, dstFormat, dec, outSep)
	if err != nil {
		return "", fmt.Errorf("amtFmtConv: errore nella conversione in output: %w", err)
	}
	return result, nil
}

// ─── lookup scala per formato ─────────────────────────────────────────────────

// numConvFormatScale restituisce il numero di cifre decimali implicite per un formato.
// Per NumFmtDecimal restituisce -1 (scala variabile).
func numConvFormatScale(format string) (int, error) {
	switch format {
	case NumFmtDecimal:
		return -1, nil
	case NumFmtInt2:
		return 2, nil
	case NumFmtInt3:
		return 3, nil
	case NumFmtInt4:
		return 4, nil
	case NumFmtInt5:
		return 5, nil
	case NumFmtInt6:
		return 6, nil
	default:
		return 0, fmt.Errorf("formato non riconosciuto: %q", format)
	}
}

// ─── parsing ─────────────────────────────────────────────────────────────────

// numConvParse converte il valore di input nella tripla canonica (neg, digits, scale).
// Dopo il parsing normalizza il "zero negativo": se il valore è zero, neg è sempre false.
func numConvParse(value interface{}, srcFormat string) (neg bool, digits string, scale int, err error) {
	srcScale, err := numConvFormatScale(srcFormat)
	if err != nil {
		return false, "", 0, fmt.Errorf("formato sorgente: %w", err)
	}
	if srcScale >= 0 {
		neg, digits, scale, err = numConvParseIntegral(value, srcFormat, srcScale)
	} else {
		neg, digits, scale, err = numConvParseDecimal(value)
	}
	if err != nil {
		return false, "", 0, err
	}
	// normalizza zero negativo: "-0", "-0.00", ecc. → neg=false
	if neg && strings.TrimLeft(digits, "0") == "" {
		neg = false
	}
	return neg, digits, scale, nil
}

// numConvParseIntegral analizza un valore atteso come numero intero con scale cifre
// decimali implicite (formato intN).
func numConvParseIntegral(value interface{}, format string, scale int) (neg bool, digits string, _ int, err error) {
	var raw string
	switch v := value.(type) {
	case int:
		raw = strconv.FormatInt(int64(v), 10)
	case int64:
		raw = strconv.FormatInt(v, 10)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false, "", 0, fmt.Errorf("valore float64 non valido (NaN o Inf) per formato %s", format)
		}
		if v != math.Trunc(v) {
			return false, "", 0, fmt.Errorf("valore float64 %g non è intero: usare %s per valori con decimali", v, NumFmtDecimal)
		}
		const safeLimit = float64(1 << 53) // massima precisione esatta di float64
		if v > safeLimit || v < -safeLimit {
			return false, "", 0, fmt.Errorf("valore float64 %g fuori dal range di rappresentazione esatta (max ±%.0f)", v, safeLimit)
		}
		raw = strconv.FormatInt(int64(v), 10)
	case string:
		raw = strings.TrimSpace(v)
		if raw == "" {
			return false, "", 0, fmt.Errorf("valore stringa vuoto non ammesso per formato %s", format)
		}
	default:
		return false, "", 0, fmt.Errorf("tipo non supportato: %T", value)
	}

	// estrai segno
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "-"):
		neg = true
		raw = strings.TrimSpace(raw[1:])
	case strings.HasPrefix(raw, "+"):
		raw = strings.TrimSpace(raw[1:])
	}
	if raw == "" {
		return false, "", 0, fmt.Errorf("valore vuoto dopo il segno per formato %s", format)
	}
	// verifica che siano tutte cifre (nessun separatore decimale nei formati interi)
	for _, c := range raw {
		if c < '0' || c > '9' {
			return false, "", 0, fmt.Errorf("formato integrale atteso ma trovato carattere non numerico %q nel valore %q", c, value)
		}
	}
	// rimuovi zeri iniziali
	raw = strings.TrimLeft(raw, "0")
	if raw == "" {
		raw = "0"
	}
	return neg, raw, scale, nil
}

// numConvParseDecimal analizza un valore in formato decimale (separatore esplicito).
func numConvParseDecimal(value interface{}) (neg bool, digits string, scale int, err error) {
	var s string
	switch v := value.(type) {
	case int:
		// strconv.FormatInt gestisce correttamente math.MinInt64 senza overflow.
		s = strconv.FormatInt(int64(v), 10)
	case int64:
		// La negazione manuale di math.MinInt64 overflowa in two's complement:
		// usiamo strconv.FormatInt che produce la stringa corretta incluso il segno.
		s = strconv.FormatInt(v, 10)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false, "", 0, fmt.Errorf("valore float64 non valido (NaN o Inf)")
		}
		// 6 cifre decimali per mantenere la precisione tipica dei valori monetari
		s = strconv.FormatFloat(v, 'f', 6, 64)
	case string:
		s = strings.TrimSpace(v)
		if s == "" {
			return false, "", 0, fmt.Errorf("valore stringa vuoto non ammesso")
		}
	default:
		return false, "", 0, fmt.Errorf("tipo non supportato: %T", value)
	}
	return numConvParseDecimalString(s)
}

// numConvParseDecimalString analizza una stringa decimale (con "." o "," come separatore)
// e restituisce la tripla canonica.
func numConvParseDecimalString(s string) (neg bool, digits string, scale int, err error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "-"):
		neg = true
		s = strings.TrimSpace(s[1:])
	case strings.HasPrefix(s, "+"):
		s = strings.TrimSpace(s[1:])
	}
	if s == "" {
		return false, "", 0, fmt.Errorf("valore vuoto dopo il segno")
	}
	// normalizza separatore decimale
	s = strings.ReplaceAll(s, ",", ".")
	// separa parte intera e decimale
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	var fracPart string
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if intPart == "" {
		intPart = "0"
	}
	for _, c := range intPart {
		if c < '0' || c > '9' {
			return false, "", 0, fmt.Errorf("parte intera non valida: carattere %q in %q", c, intPart)
		}
	}
	for _, c := range fracPart {
		if c < '0' || c > '9' {
			return false, "", 0, fmt.Errorf("parte decimale non valida: carattere %q in %q", c, fracPart)
		}
	}
	// rimuovi zeri iniziali dalla parte intera (mantieni almeno "0")
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	// digits = intPart concatenato a fracPart; scale = numero di cifre della parte decimale
	digits = intPart + fracPart
	scale = len(fracPart)
	return neg, digits, scale, nil
}

// ─── conversione di scala ─────────────────────────────────────────────────────

// numConvAdjustScale porta digits dalla scala srcScale alla scala dstScale.
//   - dstScale > srcScale: aggiunge (dstScale-srcScale) zeri a destra
//   - dstScale < srcScale: rimuove (srcScale-dstScale) cifre a destra (troncamento)
//
// Restituisce sempre una stringa non vuota di cifre decimali.
func numConvAdjustScale(digits string, srcScale, dstScale int) string {
	switch {
	case dstScale > srcScale:
		return digits + strings.Repeat("0", dstScale-srcScale)
	case dstScale < srcScale:
		trim := srcScale - dstScale
		if trim >= len(digits) {
			return "0"
		}
		result := digits[:len(digits)-trim]
		if result == "" {
			return "0"
		}
		return result
	default:
		return digits
	}
}

// ─── formattazione output ─────────────────────────────────────────────────────

// numConvFormat produce la stringa di output nel formato destinazione richiesto.
func numConvFormat(neg bool, digits string, scale int, dstFormat string, decimals int, outSep byte) (string, error) {
	dstScale, err := numConvFormatScale(dstFormat)
	if err != nil {
		return "", fmt.Errorf("formato destinazione: %w", err)
	}

	sign := ""
	if neg {
		sign = "-"
	}

	if dstScale >= 0 {
		// output intero (intN): adatta la scala e restituisce il numero senza punto decimale
		adjusted := numConvAdjustScale(digits, scale, dstScale)
		adjusted = strings.TrimLeft(adjusted, "0")
		if adjusted == "" {
			adjusted = "0"
		}
		return sign + adjusted, nil
	}

	// output decimale
	if decimals < -1 {
		decimals = -1
	}
	if decimals > 12 {
		decimals = 12
	}

	// decimals == -1 → precisione naturale: usa la scala sorgente e taglia gli zeri finali
	if decimals == -1 {
		// digits ha già la scala "scale"; la splittiamo in parte intera e decimale
		n := len(digits) - scale
		var intPart, fracPart string
		if n > 0 {
			intPart = digits[:n]
			fracPart = digits[n:]
		} else {
			// tutti i digits sono parte decimale (valore < 1)
			intPart = "0"
			fracPart = strings.Repeat("0", -n) + digits
		}
		intPart = strings.TrimLeft(intPart, "0")
		if intPart == "" {
			intPart = "0"
		}
		// elimina zeri finali non significativi dalla parte decimale
		fracPart = strings.TrimRight(fracPart, "0")
		if fracPart == "" {
			return sign + intPart, nil
		}
		return sign + intPart + string(outSep) + fracPart, nil
	}

	adjusted := numConvAdjustScale(digits, scale, decimals)

	if decimals == 0 {
		adjusted = strings.TrimLeft(adjusted, "0")
		if adjusted == "" {
			adjusted = "0"
		}
		return sign + adjusted, nil
	}

	// dividi in parte intera e parte decimale
	n := len(adjusted) - decimals
	var intPart, fracPart string
	if n > 0 {
		intPart = adjusted[:n]
		fracPart = adjusted[n:]
	} else {
		// la parte intera è zero; la parte decimale va completata con zeri a sinistra
		intPart = "0"
		fracPart = strings.Repeat("0", -n) + adjusted
	}
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	return sign + intPart + string(outSep) + fracPart, nil
}
