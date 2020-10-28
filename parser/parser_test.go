package parser

import "testing"

func Test_parseArgumentComment(t *testing.T) {
	tests := []struct {
		test             string
		line             string
		wantName         string
		wantAlias        string
		wantHasDefault   bool
		wantDefaultValue string
		wantDescription  string
	}{
		{
			test:     "should parse only name",
			line:     "var",
			wantName: "var",
		},
		{
			test:     "should parse only name with spaces",
			line:     " var ",
			wantName: "var",
		},
		{
			test:      "should parse name with alias",
			line:      "var(alias)",
			wantName:  "var",
			wantAlias: "alias",
		},
		{
			test:      "should parse name with alias with spaces",
			line:      "var ( alias )",
			wantName:  "var",
			wantAlias: "alias",
		},
		{
			test:             "should parse name with alias and default",
			line:             "var(alias)=default",
			wantName:         "var",
			wantAlias:        "alias",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name with alias and default with spaces",
			line:             "var(alias) = default",
			wantName:         "var",
			wantAlias:        "alias",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name with alias and quoted default",
			line:             "var(alias)=`default`",
			wantName:         "var",
			wantAlias:        "alias",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name with alias and quoted default with spaces",
			line:             "var(alias)= `defa ult ` ",
			wantName:         "var",
			wantAlias:        "alias",
			wantHasDefault:   true,
			wantDefaultValue: "defa ult ",
		},
		{
			test:             "should parse name with alias, quoted default with spaces and description",
			line:             "var(alias)= `defa ult `     description  ",
			wantName:         "var",
			wantAlias:        "alias",
			wantHasDefault:   true,
			wantDefaultValue: "defa ult ",
			wantDescription:  "description",
		},
		{
			test:            "should parse name and description",
			line:            "var description",
			wantName:        "var",
			wantHasDefault:  false,
			wantDescription: "description",
		},
		{
			test:             "should parse name and default",
			line:             "var=default",
			wantName:         "var",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name and default and description",
			line:             "var=default",
			wantName:         "var",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name and quoted default",
			line:             "var=`default`",
			wantName:         "var",
			wantHasDefault:   true,
			wantDefaultValue: "default",
		},
		{
			test:             "should parse name and quoted default and description",
			line:             "var=`default` description",
			wantName:         "var",
			wantHasDefault:   true,
			wantDefaultValue: "default",
			wantDescription:  "description",
		},
		{
			test:            "should parse name and alias and description",
			line:            "var(alias) description",
			wantName:        "var",
			wantAlias:       "alias",
			wantDescription: "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			gotName, gotAlias, gotHasDefault, gotDefaultValue, gotDescription := parseArgumentComment(tt.line)
			if gotName != tt.wantName {
				t.Errorf("parseArgumentComment() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotAlias != tt.wantAlias {
				t.Errorf("parseArgumentComment() gotAlias = %v, want %v", gotAlias, tt.wantAlias)
			}
			if gotHasDefault != tt.wantHasDefault {
				t.Errorf("parseArgumentComment() gotHasDefault = %v, want %v", gotHasDefault, tt.wantHasDefault)
			}
			if gotDefaultValue != tt.wantDefaultValue {
				t.Errorf("parseArgumentComment() gotDefaultValue = %v, want %v", gotDefaultValue, tt.wantDefaultValue)
			}
			if gotDescription != tt.wantDescription {
				t.Errorf("parseArgumentComment() gotDefaultValue = %v, want %v", gotDefaultValue, tt.wantDefaultValue)
			}
		})
	}
}

func Test_parseCommentType(t *testing.T) {
	tests := []struct {
		test string
		line string
		want string
	}{
		{
			test: "should detect return",
			line: "return result",
			want: "return",
		},
		{
			test: "should detect return without description",
			line: "return",
			want: "return",
		},
		{
			test: "should detect error",
			line: "0 description",
			want: "error",
		},
		{
			test: "should detect error with negative code",
			line: "-100 description",
			want: "error",
		},
		{
			test: "should detect error without description",
			line: "-100",
			want: "error",
		},
		{
			test: "should detect argument",
			line: "var(alias)",
			want: "argument",
		},
		{
			test: "should detect argument with numbers",
			line: "var100=100 description",
			want: "argument",
		},
	}
	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			if got := parseCommentType(tt.line); got != tt.want {
				t.Errorf("parseCommentType() = %v, want %v", got, tt.want)
			}
		})
	}
}
