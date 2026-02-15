package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// ReadAllParsed is [csv.Reader.ReadAll] except it parses each row into the given record struct.
// Each property in record should specify a csv tag denoting its column header, otherwise it will be ignored.
func ReadAllParsed[Record any](r io.Reader, record Record) ([]Record, error) {
	recordType := reflect.TypeOf(record)
	out := []Record{}

	// Only accept structs
	if recordType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("row must be of type struct: received %s", recordType.Kind())
	}

	csvReader := csv.NewReader(r)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read all of CSV: %v", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("CSV must not have length of 0")
	}

	// Map header to column number
	headers := records[0]
	headerToCol := make(map[string]int)
	for i, header := range headers {
		headerToCol[header] = i
	}

	// Create new records
	for _, record := range records[1:] {
		newRecord := reflect.New(recordType).Elem()

		for i := 0; i < recordType.NumField(); i++ {
			fieldValue := newRecord.Field(i)
			fieldType := newRecord.Type().Field(i)

			// Get CSV header from tag or skip
			header := fieldType.Tag.Get("csv")
			if header == "" {
				continue
			}

			// Parse and set field
			if col, exists := headerToCol[header]; exists && col < len(record) {
				field := record[col]

				switch fieldType.Type.Kind() {
				case reflect.String:
					val := strings.Trim(field, "\"")
					fieldValue.SetString(val)
				case reflect.Int:
					val, err := strconv.Atoi(field)
					if err != nil && field != "" {
						return nil, fmt.Errorf("failed to parse field %s: %v", fieldType.Name, err)
					}
					fieldValue.SetInt(int64(val))
				case reflect.Int64:
					val, err := strconv.ParseInt(field, 10, 64)
					if err != nil && field != "" {
						return nil, fmt.Errorf("failed to parse field %s: %v", fieldType.Name, err)
					}
					fieldValue.SetInt(val)
				case reflect.Bool:
					val, err := strconv.ParseBool(field)
					if err != nil {
						return nil, fmt.Errorf("failed to parse field %s: %v", fieldType.Name, err)
					}
					fieldValue.SetBool(val)
				case reflect.Float64:
					val, err := strconv.ParseFloat(field, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse field %s: %v", fieldType.Name, err)
					}
					fieldValue.SetFloat(val)
				default:
					return nil, fmt.Errorf("unsupported type %v for field %s", fieldType.Type.Kind(), fieldType.Name)
				}
			}
		}

		out = append(out, newRecord.Interface().(Record))
	}

	return out, nil
}
