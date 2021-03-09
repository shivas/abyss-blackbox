package combatlog

import (
	"reflect"
	"testing"
	"time"
)

func TestReader_GetLogFiles(t *testing.T) {
	type args struct {
		since      time.Time
		timeWindow time.Duration
	}

	r := NewReader("./testdata")

	tests := []struct {
		name    string
		r       *Reader
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "same day",
			r:    r,
			args: args{since: time.Date(2020, 10, 23, 0, 12, 13, 0, time.UTC), timeWindow: 23 * time.Hour},
			want: []string{
				"testdata\\20201022_205944.txt",
				"testdata\\20201022_205944_french.txt",
				"testdata\\20201022_205944_german.txt",
				"testdata\\20201022_205944_korean.txt",
				"testdata\\20201022_222559.txt",
				"testdata\\20201022_222614.txt",
				"testdata\\20201022_222614_newer.txt",
			},
		},
		// TODO: Add test cases.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetLogFiles(tt.args.since, tt.args.timeWindow)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reader.GetCharacters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reader.GetCharacters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_genPrefixes(t *testing.T) {
	type args struct {
		end        time.Time
		timeWindow time.Duration
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "same day",
			args: args{end: time.Date(2020, 10, 23, 0, 12, 13, 0, time.UTC), timeWindow: 23 * time.Hour},
			want: []string{"20201022"},
		},
		{
			name: "one day window",
			args: args{end: time.Date(2020, 10, 23, 0, 12, 13, 0, time.UTC), timeWindow: 24 * time.Hour},
			want: []string{"20201022", "20201023"},
		},
		{
			name: "two days window",
			args: args{end: time.Date(2020, 10, 23, 0, 12, 13, 0, time.UTC), timeWindow: 48 * time.Hour},
			want: []string{"20201021", "20201022", "20201023"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genPrefixes(tt.args.end, tt.args.timeWindow); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("genPrefixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReader_MapCharactersToFiles(t *testing.T) {
	type args struct {
		files []string
	}

	r := NewReader("./testdata")

	tests := []struct {
		name string
		r    *Reader
		args args
		want map[string]CombatLogFile
	}{
		{
			name: "first",
			args: args{
				files: []string{
					"testdata\\20201022_205944.txt",
					"testdata\\20201022_222559.txt",
					"testdata\\20201022_222614.txt",
					"testdata\\20201022_222614_newer.txt",
					"testdata\\20201022_205944_french.txt",
					"testdata\\20201022_205944_korean.txt",
					"testdata\\20201022_205944_german.txt",
				},
			},
			r: r,
			want: map[string]CombatLogFile{
				"Runner2": {Filename: "testdata\\20201022_222614_newer.txt", LanguageCode: LanguageCode_ENGLISH},
				"Runner1": {Filename: "testdata\\20201022_205944.txt", LanguageCode: LanguageCode_ENGLISH},
				"French":  {Filename: "testdata\\20201022_205944_french.txt", LanguageCode: LanguageCode_FRENCH},
				"Korean":  {Filename: "testdata\\20201022_205944_korean.txt", LanguageCode: LanguageCode_KOREAN},
				"German":  {Filename: "testdata\\20201022_205944_german.txt", LanguageCode: LanguageCode_GERMAN},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.MapCharactersToFiles(tt.args.files); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reader.MapCharactersToFiles() = %v, want %v", got, tt.want)
			} else {
				t.Logf("%+v\n", got)
			}
		})
	}
}
