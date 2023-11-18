package core

import (
	"fmt"
	"testing"
)

type SessionTest struct {
	Name string
}

func (s *SessionTest) Start(args *SessionConfig) error {
	fmt.Println("Test Start")
	return nil
}

func (s *SessionTest) Stop() error {
	fmt.Println("Test Stop")
	return nil
}

func (s *SessionTest) ReadData(data any) error {
	fmt.Println("Test DataRead")
	return nil
}

func (s *SessionTest) ExportData(data any) error {
	fmt.Println("Test DataExport")
	return nil
}

// https://stackoverflow.com/questions/40823315/x-does-not-implement-y-method-has-a-pointer-receiver
var testMap = map[string]Spy{
	"test": &SessionTest{Name: "test"},
}

func TestSession(*testing.T) {
	for name, session := range testMap {
		fmt.Println("name:", name)
		session.Start(nil)
	}
}
