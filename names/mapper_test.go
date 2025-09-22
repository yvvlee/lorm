package names

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSameMapper(t *testing.T) {
	mapper := SameMapper{}
	testCases := []struct {
		input    string
		expected string
	}{
		{"UserName", "UserName"},
		{"ID", "ID"},
		{"user_name", "user_name"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapper.ConvertName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSnakeMapper(t *testing.T) {
	mapper := SnakeMapper{}
	testCases := []struct {
		input    string
		expected string
	}{
		{"UserName", "user_name"},
		{"ID", "id"},
		{"user_name", "user_name"},
		{"UserNameABC", "user_name_abc"},
		{"", ""},
		{"A", "a"},
		{"HTML", "html"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapper.ConvertName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCamelMapper(t *testing.T) {
	mapper := CamelMapper{}
	testCases := []struct {
		input    string
		expected string
	}{
		{"user_name", "userName"},
		{"UserName", "userName"},
		{"id", "id"},
		{"ID", "id"},
		{"user_name_abc", "userNameAbc"},
		{"", ""},
		{"a", "a"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapper.ConvertName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPrefixMapper(t *testing.T) {
	baseMapper := SnakeMapper{}
	mapper := NewPrefixMapper(baseMapper, "prefix_")

	testCases := []struct {
		input    string
		expected string
	}{
		{"UserName", "prefix_user_name"},
		{"ID", "prefix_id"},
		{"", "prefix_"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapper.ConvertName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSuffixMapper(t *testing.T) {
	baseMapper := SnakeMapper{}
	mapper := NewSuffixMapper(baseMapper, "_suffix")

	testCases := []struct {
		input    string
		expected string
	}{
		{"UserName", "user_name_suffix"},
		{"ID", "id_suffix"},
		{"", "_suffix"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := mapper.ConvertName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
