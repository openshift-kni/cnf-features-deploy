package yamltrim

import "log"

// YamlTrim will deeply trim a given structure:
// - For maps (only map[string]interface{} supported for now), recursively remove any entries whose values are zero, and return 'nil' if the final result is an empty map
// - For slices (only []interface{} supported for now), recursively remove any entries that are zero, and return 'nil' if the final result is an empty slice
// - For scalar types, return 'nil' for a zero value, else return the value
func YamlTrim(src interface{}) interface{} {
	switch t := src.(type) {
	case map[string]interface{}:
		// Recursrvely check the next level down
		t = trimMap(t)
		// only retain if the trimmed result has content
		if len(t) > 0 {
			return t
		}
		return nil
	case []interface{}:
		// Recursively check all items in the slice
		t = trimSlice(t)
		// only retain if the trimmed result has content
		if len(t) > 0 {
			return t
		}
		return nil
	case string:
		// omit empty strings
		if len(t) == 0 {
			return nil
		}
	case bool:
		// omit false booleans
		if !t {
			return nil
		}
	case int:
		// omit zeroes
		if t == 0 {
			return nil
		}
	case float64:
		// omit zeroes
		if t == 0.0 {
			return nil
		}
	case nil:
		// omit nil pointers
		return nil
	default:
		// Report but retain everything else
		log.Printf("Unknown type: %v (%T)\n", src, src)
	}
	return src
}

func trimSlice(src []interface{}) []interface{} {
	var dst []interface{}
	for _, v := range src {
		t := YamlTrim(v)
		if t != nil {
			dst = append(dst, t)
		}
	}
	return dst
}

func trimMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		t := YamlTrim(v)
		if t != nil {
			dst[k] = t
		}
	}
	return dst
}
