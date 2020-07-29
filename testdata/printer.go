package testdata

import "github.com/semrush/zenrpc/v2"

type PrintService struct{ zenrpc.Service }

//zenrpc:s="test"
func (PrintService) PrintRequiredDefault(s string) string {
	return s
}

//zenrpc:s="test"
func (PrintService) PrintOptionalWithDefault(s *string) string {
	// if client passes nil to this method it will be replaced with default value
	return *s
}

func (PrintService) PrintRequired(s string) string {
	return s
}

func (PrintService) PrintOptional(s *string) string {
	if s == nil {
		return "string is empty"
	}

	return *s
}
