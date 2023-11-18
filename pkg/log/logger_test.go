package log

import "testing"

func TestLoger(*testing.T) {
	//loger := NewLogger()
	//loger.logger.SetOutput(loger.file)
	//loger.logger.SetLevel(loger.level)
	Loger.Info("Info test")
	Loger.Error("Error test")
	Loger.Warn("Warn test")
	Loger.Debug("Debug test")
}
