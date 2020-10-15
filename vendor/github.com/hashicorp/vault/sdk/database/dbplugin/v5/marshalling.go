package dbplugin

import (
	"math"

	"google.golang.org/protobuf/types/known/structpb"
)

func mapToStruct(m map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(m)
}

func structToMap(strct *structpb.Struct) map[string]interface{} {
	m := strct.AsMap()
	coerceFloatsToInt(m)
	return m
}

// coerceFloatsToInt if the floats can be coerced to an integer without losing data
func coerceFloatsToInt(m map[string]interface{}) {
	for k, v := range m {
		fVal, ok := v.(float64)
		if !ok {
			continue
		}
		if isInt(fVal) {
			m[k] = int64(fVal)
		}
	}
}

// isInt attempts to determine if the given floating point number could be represented as an integer without losing data
// This does not work for very large floats, however in this usage that's okay since we don't expect numbers that large.
func isInt(f float64) bool {
	return math.Floor(f) == f
}
