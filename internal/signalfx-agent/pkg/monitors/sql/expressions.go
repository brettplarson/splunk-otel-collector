package sql

import (
	"database/sql"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"reflect"
)

type ExprEnv map[string]interface{}

func newExprEnv(rowSlice []interface{}, columnNames []string) ExprEnv {
	out := make(map[string]interface{}, len(columnNames))

	values := scannedToValues(rowSlice)
	for i, name := range columnNames {
		// It is expected that the caller has the lengths matched properly or
		// else this will panic.
		out[name] = values[i]
	}

	return ExprEnv(out)
}

func scannedToValues(scanners []interface{}) []interface{} {
	vals := make([]interface{}, len(scanners))

	for i := range scanners {
		switch s := scanners[i].(type) {
		case *sql.NullString:
			if s.Valid {
				vals[i] = s.String
			}
		case *sql.NullFloat64:
			if s.Valid {
				vals[i] = s.Float64
			}
		case *sql.NullTime:
			if s.Valid {
				vals[i] = s.Time
			}
		case *sql.NullBool:
			if s.Valid {
				vals[i] = s.Bool
			}
		case *interface{}:
			if s == nil {
				continue
			}
			switch s2 := (*s).(type) {
			case []uint8:
				vals[i] = string(s2)
			case *[]uint8:
				vals[i] = string(*s2)
			}
		default:
			vals[i] = scanners[i]
		}
	}

	return vals
}

var floatType = reflect.TypeOf(float64(0))

func convertToFloatOrPanic(val interface{}) float64 {
	rVal := reflect.Indirect(reflect.ValueOf(val))
	if !rVal.IsValid() {
		panic("value is null")
	}

	if !rVal.Type().ConvertibleTo(floatType) {
		// expr will recover from panics and return them as errors when
		// evaluating the expression.
		panic("value must be float")
	}

	return rVal.Convert(floatType).Float()
}

func (e ExprEnv) GAUGE(metric string, dims map[string]interface{}, val interface{}) pmetric.Metric {
	m := pmetric.NewMetric()
	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(convertToFloatOrPanic(val))
	dp.Attributes().FromRaw(dims)
	m.SetName(metric)
	return m
}

func (e ExprEnv) CUMULATIVE(metric string, dims map[string]interface{}, val interface{}) pmetric.Metric {
	m := pmetric.NewMetric()
	dp := m.SetEmptySum().DataPoints().AppendEmpty()
	dp.SetDoubleValue(convertToFloatOrPanic(val))
	dp.Attributes().FromRaw(dims)
	m.SetName(metric)
	return m
}
