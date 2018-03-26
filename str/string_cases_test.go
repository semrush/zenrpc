package str

import (
	"testing"
)

var testCases = []string{
	"stringService",
	"StringService",
	"Stringservice",
	"string_service",
	"String_service",
	"string_Service",
	"JSONService",
	"jsonService",
	"JSONServiceV2",
	"JSONService V2",
}

func Test_toSnakeCase(t *testing.T) {
	answers := []string{
		"string_service",
		"string_service",
		"stringservice",
		"string_service",
		"string_service",
		"string_service",
		"json_service",
		"json_service",
		"json_service_v2",
		"json_service_v2",
	}
	if len(testCases) != len(answers) {
		t.Fatal("different amount of test cases and expected answers")
	}
	for i, tt := range testCases {
		t.Run(tt, func(t *testing.T) {
			if got := ToSnakeCase(tt); got != answers[i] {
				t.Errorf("ToSnakeCase() = %v, want %v", got, answers[i])
			}
		})
	}
}

func Test_ToURLSnakeCase(t *testing.T) {
	answers := []string{
		"string-service",
		"string-service",
		"stringservice",
		"string_service",
		"string_service",
		"string_-service",
		"json-service",
		"json-service",
		"json-service-v2",
		"json-service-v2",
	}
	if len(testCases) != len(answers) {
		t.Fatal("different amount of test cases and expected answers")
	}
	for i, tt := range testCases {
		t.Run(tt, func(t *testing.T) {
			if got := ToURLSnakeCase(tt); got != answers[i] {
				t.Errorf("ToURLSnakeCase() = %v, want %v", got, answers[i])
			}
		})
	}
}
