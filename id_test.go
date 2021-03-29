package zenrpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRawMessage(b []byte) *json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	rm := json.RawMessage(b)
	return &rm
}

func Test_newID(t *testing.T) {
	type args struct {
		rawID *json.RawMessage
	}
	tests := []struct {
		name    string
		args    args
		want    ID
		wantErr bool
	}{
		{
			name: "int",
			args: args{
				rawID: getRawMessage([]byte(`25`)),
			},
			want: ID{
				Int:   25,
				State: IDStateInt,
			},
		},
		{
			name: "string",
			args: args{
				rawID: getRawMessage([]byte(`"25"`)),
			},
			want: ID{
				String: "25",
				State:  IDStateString,
			},
		},
		{
			name: "float",
			args: args{
				rawID: getRawMessage([]byte(`25.25`)),
			},
			want: ID{
				Float: 25.25,
				State: IDStateFloat,
			},
		},
		{
			name: "null",
			args: args{
				rawID: nil,
			},
			want: ID{
				State: IDStateNull,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newID(tt.args.rawID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
