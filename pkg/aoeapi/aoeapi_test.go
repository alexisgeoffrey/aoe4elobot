package aoeapi

import (
	"reflect"
	"testing"
)

func TestQueryAll(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryAll(tt.args.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QueryAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryToMap(t *testing.T) {
	type args struct {
		data      Payload
		matchType string
		sm        safeMap
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := queryToMap(tt.args.data, tt.args.matchType, tt.args.sm); (err != nil) != tt.wantErr {
				t.Errorf("queryToMap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	type args struct {
		data      Payload
		matchType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Query(tt.args.data, tt.args.matchType)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Query() = %v, want %v", got, tt.want)
			}
		})
	}
}
