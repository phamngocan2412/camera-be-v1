package i18n

import (
	"testing"
)

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name       string
		lang       string
		key        string
		want       string
	}{
		{
			name: "English invalid credentials",
			lang: "en",
			key:  "invalid_credentials",
			want: "Invalid email or password",
		},
		{
			name: "Vietnamese invalid credentials",
			lang: "vi",
			key:  "invalid_credentials",
			want: "Email hoặc mật khẩu không đúng",
		},
		{
			name: "Default language invalid credentials",
			lang: "fr",
			key:  "invalid_credentials",
			want: "Invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessage(tt.lang, tt.key); got != tt.want {
				t.Errorf("GetMessage(%v, %v) = %v, want %v", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}
