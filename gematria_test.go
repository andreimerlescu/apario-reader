package main

import (
	`reflect`
	`testing`
)

func TestNewGemScore(t *testing.T) {
	type args struct {
		data string
	}
	type testStruct struct {
		name string
		args args
		want GemScore
	}
	tests := []testStruct{
		testStruct{
			"manifesting three six nine",
			args{
				"manifesting three six nine",
			},
			GemScore{
				Jewish:  1028,
				English: 1602,
				Simple:  267,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGemScore(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGemScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
