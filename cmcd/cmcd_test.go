package cmcd

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
)

func ExampleParseInfo() {
	ex := "http://test.example.com/?CMCD=br%3D3200%2Cbs%2Cd%3D4004%2Cmtp%3D25400"
	u, err := url.Parse(ex)
	param := u.Query().Get("CMCD")
	fmt.Println(param)
	info, err := ParseInfo(param)
	if err != nil {
		// handle...
	}
	fmt.Println(info.Bitrate)
	// Output:
	// br=3200,bs,d=4004,mtp=25400
	// 3200
}

func Test_parseRange(t *testing.T) {
	tests := []struct {
		args    string
		want    Range
		wantErr bool
	}{
		{
			args:    "0-",
			want:    [2]int{0, -1},
			wantErr: false,
		},
		{
			args:    "-0",
			want:    [2]int{-1, 0},
			wantErr: false,
		},
		{
			args:    "10-100",
			want:    [2]int{10, 100},
			wantErr: false,
		},
		{
			args:    "111",
			want:    [2]int{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := parseRange(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRange() = %v, want %v", got, tt.want)
			}
		})
	}
}
