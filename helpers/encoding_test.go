package helpers

import "testing"

func TestNormalizeByLangRepairsCommonUTF8Mojibake(t *testing.T) {
	tests := []struct {
		name  string
		input string
		lang  string
		want  string
	}{
		{
			name:  "French",
			input: "FranÃ§ais",
			lang:  "fr",
			want:  "Français",
		},
		{
			name:  "Spanish",
			input: "JosÃ©",
			lang:  "es",
			want:  "José",
		},
		{
			name:  "Russian",
			input: "\u00d0\u009c\u00d0\u00be\u00d1\u0081\u00d0\u00ba\u00d0\u00b2\u00d0\u00b0",
			lang:  "ru",
			want:  "Москва",
		},
		{
			name:  "Korean",
			input: "\u00ec\u0084\u009c\u00ec\u009a\u00b8",
			lang:  "ko",
			want:  "서울",
		},
		{
			name:  "Chinese",
			input: "\u00e4\u00b8\u00ad\u00e6\u0096\u0087",
			lang:  "zh",
			want:  "中文",
		},
		{
			name:  "Japanese",
			input: "\u00e3\u0083\u00a8\u00e3\u0082\u00ac",
			lang:  "ja",
			want:  "ヨガ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeByLang(tc.input, tc.lang)
			if got != tc.want {
				t.Fatalf("NormalizeByLang(%q, %q) = %q, want %q", tc.input, tc.lang, got, tc.want)
			}
		})
	}
}

func TestNormalizeByLangPreservesValidUnicode(t *testing.T) {
	input := "相戸愛"
	got := NormalizeByLang(input, "ja")
	if got != input {
		t.Fatalf("NormalizeByLang changed valid unicode: got %q want %q", got, input)
	}
}
