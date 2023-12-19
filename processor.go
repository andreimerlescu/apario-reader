package main

import (
	`context`
	`log`
	`sync`
)

func ProcessRow(headerFields []string, rowFields []string, rowWg *sync.WaitGroup, row chan []Column) {
	defer rowWg.Done()
	var d = map[string]string{}
	if len(headerFields) != len(rowFields) {
		if len(headerFields) < len(rowFields) {
			for i, r := range rowFields {
				if i >= len(headerFields) || len(r) == 0 {
					continue
				}
				d[headerFields[i]] = r
			}
		} else {
			for i, h := range headerFields {
				if i >= len(rowFields) || len(h) == 0 {
					continue
				}
				d[h] = rowFields[i]
			}
		}
	}
	var rowData = []Column{}
	if len(d) > 0 {
		for h, v := range d {
			rowData = append(rowData, Column{Header: h, Value: v})
		}
	} else {
		for i := 0; i < len(rowFields); i++ {
			value := rowFields[i]
			if i == 0 && len(value) == 0 {
				return
			}
			if len(headerFields) < i {
				log.Printf("skipping rowField %v due to headerFields not matching up properly", rowFields[i])
				continue
			}
			rowData = append(rowData, Column{headerFields[i], value})
		}
	}
	row <- rowData
}

func ReceiveRows(ctx context.Context, row chan []Column, filename string, callback CallbackFunc, done chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case populatedRow, ok := <-row:
			if !ok {
				done <- struct{}{}
				return
			}
			ctx := context.WithValue(ctx, CtxKey("csv_file"), filename)
			callbackErr := callback(ctx, populatedRow)
			if callbackErr != nil {
				log.Printf("failed to insert row %v with error %v", populatedRow, callbackErr)
			}
		}
	}
}
