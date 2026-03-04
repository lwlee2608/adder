package adder

import (
	"encoding/json"
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
  "APIKey": "A*****IJ",
  "Auth": {
    "Password": "*****",
    "Token": "*****ken",
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
  "Secret": "*****"
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
  "Secret": "*****"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_InvalidTagPartInvalidatesWholeRule(t *testing.T) {
	type config struct {
		Secret string `mask:"first=3,bogus=x"`
	}

	v := config{Secret: "abcdef"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "*****"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_OverlappingFirstLastFallsBackToFullMask(t *testing.T) {
	type config struct {
		Secret string `mask:"first=2,last=2"`
	}

	v := config{Secret: "abcd"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "*****"
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

func TestPrettyJSON_UnicodeMaskingIsRuneAware(t *testing.T) {
	type config struct {
		Secret string `mask:"first=2"`
	}

	v := config{Secret: "日本語abc"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "日本*****"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_PreserveLengthOption(t *testing.T) {
	type config struct {
		Secret string `mask:"first=1,last=1,preserve=true"`
	}

	v := config{Secret: "ABCDEFGHIJ"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "A********J"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_PreserveLengthOptionOnFullMask(t *testing.T) {
	type config struct {
		Secret string `mask:"true,preserve=true"`
	}

	v := config{Secret: "abcdef"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "******"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_MaskFalseSkipsMasking(t *testing.T) {
	type config struct {
		Secret string `mask:"false"`
	}

	v := config{Secret: "abcdef"}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": "abcdef"
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_EmptyStringStaysEmpty(t *testing.T) {
	type config struct {
		Secret string `mask:"true"`
	}

	v := config{Secret: ""}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	want := `{
  "Secret": ""
}`
	assert.Equal(t, want, got)
}

func TestPrettyJSON_MasksSlicesAndMaps(t *testing.T) {
	type auth struct {
		Secret string `mask:"true"`
	}

	type config struct {
		List []auth
		ByID map[string]auth
	}

	v := config{
		List: []auth{{Secret: "first"}},
		ByID: map[string]auth{"a": {Secret: "second"}},
	}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	var decoded config
	require.NoError(t, json.Unmarshal([]byte(got), &decoded))
	require.Len(t, decoded.List, 1)
	require.Contains(t, decoded.ByID, "a")
	assert.Equal(t, "*****", decoded.List[0].Secret)
	assert.Equal(t, "*****", decoded.ByID["a"].Secret)

	assert.Equal(t, "first", v.List[0].Secret)
	assert.Equal(t, "second", v.ByID["a"].Secret)
}

func TestPrettyJSON_MasksPointerToPointerStruct(t *testing.T) {
	type secret struct {
		Value string `mask:"last=2"`
	}

	type config struct {
		Ref **secret
	}

	inner := &secret{Value: "abcdef"}
	p := &inner
	v := config{Ref: p}

	got, err := PrettyJSON(v)
	require.NoError(t, err)

	var decoded struct {
		Ref *secret
	}
	require.NoError(t, json.Unmarshal([]byte(got), &decoded))
	require.NotNil(t, decoded.Ref)
	assert.Equal(t, "*****ef", decoded.Ref.Value)
	assert.Equal(t, "abcdef", (**v.Ref).Value)
}
