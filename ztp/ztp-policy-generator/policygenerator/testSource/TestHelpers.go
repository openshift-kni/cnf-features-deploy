package testSource

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"reflect"
	"testing"
)

func readFileToMap(filePath string, t *testing.T) map[string]interface{} {
	fileData := make(map[string]interface{})
	file1, err := ioutil.ReadFile(filePath)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	err = yaml.Unmarshal(file1, &fileData)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	return fileData
}

func substituteMapData(sourceMap map[string]interface{}, valueMap map[string]interface{}) map[string]interface{} {
	for key, value := range valueMap {
		if reflect.TypeOf(value).Kind() == reflect.Map {
			sourceMap[key] = substituteMapData(sourceMap[key].(map[string]interface{}), value.(map[string]interface{}))
		} else if reflect.TypeOf(value).Kind() == reflect.Slice {
			valueArr := value.([]interface{})
			sourceMapArr := make([]interface{}, 1)
			for i := 0; i < len(valueArr); i++ {
				sourceMapArr[i] = substituteMapData(sourceMap[key].([]interface{})[i].(map[string]interface{}), valueArr[i].(map[string]interface{}))
			}
			sourceMap[key] = sourceMapArr

		} else {
			sourceMap[key] = value
		}
	}
	for key, value := range sourceMap {
		if value == nil ||
			(value != nil && reflect.ValueOf(value).Kind() == reflect.String &&
				(value.(string) == "" || value.(string) == NOT_APPLICABLE)) {
			delete(sourceMap, key)
		}
	}
	return sourceMap
}
