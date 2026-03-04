package adder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrettyJSON_MasksTaggedFields(t *testing.T) {
	type authConfig struct {
		Password string `mask:"true"`
		Token    string `mask:"last=3"`
		Note     string
	}

	type config struct {
		Name   string
		APIKey string `mask:"first=1,last=2"`
		Auth   authConfig
	}

	v := config{
		Name:   "service",
		APIKey: "ABCDEFGHIJ",
		Auth: authConfig{
			Password: "s3cret",
			Token:    "secretToken",
			Note:     "visible",
		},
	}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Name": "service",
  "APIKey": "A*******IJ",
  "Auth": {
    "Password": "******",
    "Token": "********ken",
    "Note": "visible"
  }
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_DoesNotMutateOriginal_PointerField(t *testing.T) {
	type dbConfig struct {
		URL string `mask:"true"`
	}

	type config struct {
		DB *dbConfig
	}

	v := config{DB: &dbConfig{URL: "postgres://secret"}}

	_, err := PrettyJSON(v)
	require.NoError(t, err)
	assert.Equal(t, "postgres://secret", v.DB.URL)
}

func TestPrettyJSON_PointerInput(t *testing.T) {
	type config struct {
		Secret string `mask:"true"`
	}

	v := &config{Secret: "abcd"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "****"
}`
	assert.Equal(t, want, got)
	assert.Equal(t, "abcd", v.Secret)
}

func TestPrettyJSON_NilPointerInput(t *testing.T) {
	type config struct {
		Secret string `mask:"true"`
	}

	var v *config

	got, err := PrettyJSON(v)
	require.NoError(t, err)
	assert.Equal(t, "null", got)
}

func TestPrettyJSON_NonStructInput(t *testing.T) {
	got, err := PrettyJSON("hello")
	require.NoError(t, err)
	assert.Equal(t, `"hello"`, got)
}

func TestPrettyJSON_InvalidMaskTagFallsBackToFullMask(t *testing.T) {
	type config struct {
		Secret string `mask:"first=abc"`
	}

	v := config{Secret: "abcdef"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "******"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_ForceAtLeastOneMaskedRune(t *testing.T) {
	type config struct {
		Secret string `mask:"first=2,last=2"`
	}

	v := config{Secret: "abcd"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "ab*d"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_MarshalError(t *testing.T) {
	type config struct {
		Ch chan int
	}

	v := config{Ch: make(chan int)}

	_, err := PrettyJSON(v)
	require.Error(t, err)
}
