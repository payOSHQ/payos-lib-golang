package payos

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/payOSHQ/payos-lib-golang/v2/internal/apierror"
)

// Non-finite numbers cannot be produced by encoding/json, so the validator is exercised directly.
func TestValidateResponseDataRejectsNonFiniteNumbers(t *testing.T) {
	cases := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{"NaN", map[string]interface{}{"amount": math.NaN()}, `"amount"`},
		{"positive infinity", map[string]interface{}{"amount": math.Inf(1)}, `"amount"`},
		{"negative infinity in list item", map[string]interface{}{
			"transactions": []interface{}{map[string]interface{}{"amount": math.Inf(-1)}},
		}, `"transactions[0].amount"`},
		{"json.Number overflow", map[string]interface{}{"amount": json.Number("1e400")}, `"amount"`},
		{"json.Number unsafe integer", map[string]interface{}{"amount": json.Number("9007199254740993")}, `"amount"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateResponseData(tc.data)
			var sigErr *apierror.InvalidSignatureError
			if !errors.As(err, &sigErr) {
				t.Fatalf("expected *apierror.InvalidSignatureError, got %T (%v)", err, err)
			}
			want := "Invalid response data: " + tc.want + " must be a finite number within the safe integer range"
			if sigErr.Message != want {
				t.Fatalf("message = %q, want %q", sigErr.Message, want)
			}
		})
	}
}

func TestValidateResponseDataRejectsNonObjectData(t *testing.T) {
	for _, data := range []interface{}{"text", []interface{}{1}, 1.0, nil} {
		err := validateResponseData(data)
		var sigErr *apierror.InvalidSignatureError
		if !errors.As(err, &sigErr) || sigErr.Message != "Invalid response data: data must be a JSON object" {
			t.Fatalf("data %#v: error = %v", data, err)
		}
	}
}
