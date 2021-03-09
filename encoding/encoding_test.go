package encoding

import (
	"bytes"
	"io"
	"testing"

	"github.com/shivas/abyss-blackbox/combatlog"
)

func TestAbyssRecording_Encode(t *testing.T) {

	record := AbyssRecording{
		Overview: []byte{1, 2, 3},
		Loot: []*LootRecord{
			{Frame: 1, Loot: "loot record1"},
			{Frame: 5, Loot: "loot record2"},
		},
		CombatLog: []*combatlog.CombatLogRecord{
			{
				CharacterName:  "runner1",
				CombatLogLines: []string{"line1", "line2"},
				LanguageCode:   combatlog.LanguageCode_ENGLISH,
			},
		},
	}

	var buf bytes.Buffer

	type args struct {
		w io.Writer
	}

	tests := []struct {
		name    string
		rf      *AbyssRecording
		args    args
		wantErr bool
	}{
		{"testcase1", &record, args{&buf}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rf.Encode(tt.args.w); (err != nil) != tt.wantErr {
				t.Errorf("AbyssRecording.Encode() error = %v, wantErr %v", err, tt.wantErr)
			}

			_, err := Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Errorf("AbyssRecording.Decode() error = %v", err)
			}
		})
	}
}
