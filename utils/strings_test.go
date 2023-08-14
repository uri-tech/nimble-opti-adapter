package utils

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIngressKey(t *testing.T) {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "test-namespace",
		},
	}

	expected := "test-namespace/test-ingress"
	got := IngressKey(ing)
	if got != expected {
		t.Errorf("IngressKey() = %v; want %v", got, expected)
	}
	t.Logf("got: %v", got)
}

func TestChangeSecretName(t *testing.T) {
	tests := []struct {
		name       string
		secretName string
		want       string
		wantErr    bool
	}{
		{"No Suffix", "my-secret", "my-secret-v1", false},
		{"Valid Suffix", "my-secret-v1", "my-secret-v2", false},
		{"Invalid Suffix", "my-secret-vx", "my-secret-vx-v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChangeSecretName(tt.secretName)
			t.Logf("got: %v", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangeSecretName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ChangeSecretName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStrHasVxSuffix(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want bool
	}{
		{"No Suffix", "my-secret", false},
		{"Valid Suffix", "my-secret-v1", true},
		{"Invalid Suffix", "my-secret-vx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStrHasVxSuffix(tt.str); got != tt.want {
				t.Errorf("isStrHasVxSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddVxSuffixToStr(t *testing.T) {
	if got := AddVxSuffixToStr("my-secret"); got != "my-secret-v1" {
		t.Errorf("AddVxSuffixToStr() = %v, want %v", got, "my-secret-v1")
	}
}

func TestIncVxSuffixToStr(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    string
		wantErr bool
	}{
		{"Valid Suffix", "my-secret-v1", "my-secret-v2", false},
		{"Valid Suffix", "my-secret-v5", "my-secret-v6", false},
		{"Valid Suffix", "my-secret-v9", "my-secret-v10", false},
		{"Invalid Suffix", "my-secret-vx", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncVxSuffixToStr(tt.str)
			t.Logf("got: %v", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("IncVxSuffixToStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IncVxSuffixToStr() = %v, want %v", got, tt.want)
			}

		})
	}
}
