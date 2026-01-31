package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChain(t *testing.T) {
	t.Parallel()

	errFirst := errors.New("first error")
	errSecond := errors.New("second error")

	tests := []struct {
		name        string
		validators  Chain
		expectError error
	}{
		{
			name:        "empty chain passes",
			validators:  Chain{},
			expectError: nil,
		},
		{
			name: "single passing validator",
			validators: Chain{
				ValidatorFunc(func() error { return nil }),
			},
			expectError: nil,
		},
		{
			name: "single failing validator",
			validators: Chain{
				ValidatorFunc(func() error { return errFirst }),
			},
			expectError: errFirst,
		},
		{
			name: "multiple passing validators",
			validators: Chain{
				ValidatorFunc(func() error { return nil }),
				ValidatorFunc(func() error { return nil }),
				ValidatorFunc(func() error { return nil }),
			},
			expectError: nil,
		},
		{
			name: "first validator fails stops chain",
			validators: Chain{
				ValidatorFunc(func() error { return errFirst }),
				ValidatorFunc(func() error { return errSecond }),
			},
			expectError: errFirst,
		},
		{
			name: "second validator fails",
			validators: Chain{
				ValidatorFunc(func() error { return nil }),
				ValidatorFunc(func() error { return errSecond }),
			},
			expectError: errSecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.validators.Validate()
			if tt.expectError != nil {
				require.ErrorIs(t, err, tt.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatorFunc(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		v := ValidatorFunc(func() error { return nil })
		require.NoError(t, v.Validate())
	})

	t.Run("failure", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("validation failed")
		v := ValidatorFunc(func() error { return expectedErr })
		require.ErrorIs(t, v.Validate(), expectedErr)
	})
}

// Integration tests would require mocking the engine and git interfaces.
// These are better tested as part of the action tests that use the validators.
