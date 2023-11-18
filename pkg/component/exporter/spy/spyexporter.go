package spyexporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chentao-kernel/spycat/pkg/component/exporter"
	"github.com/chentao-kernel/spycat/pkg/core/model"
)

type SpyExporter struct {
	config   *Config
	outputer Outputer
}

type Outputer interface {
	output(data *model.DataBlock) error
	name() string
}

func NewSpyExporter(config interface{}) exporter.Exporter {
	var outer Outputer
	cfg, _ := config.(*Config)

	if cfg.OutPuter == "FileOutpter" {
		outer = &FileOutputer{
			Name: cfg.OutPuter,
			config: &Config{
				BaseFilePath: "/tmp/",
			},
			file: nil,
		}
	} else if cfg.OutPuter == "TerminalOutper" {
		outer = &TerminalOutputer{
			Name:   cfg.OutPuter,
			config: &Config{},
			file:   nil,
		}
	}

	se := &SpyExporter{
		config:   cfg,
		outputer: outer,
	}
	return se
}

func (s *SpyExporter) Consume(data *model.DataBlock) error {
	s.outputer.output(data)
	return nil
}

type FileOutputer struct {
	Name   string
	config *Config
	file   *os.File
}

func (f *FileOutputer) output(data *model.DataBlock) error {
	if f.file == nil {
		comm := data.Labels.GetStringValue(model.Comm)
		pid := data.Labels.GetIntValue(model.Pid)
		fileName := comm + "_" + string(pid)
		filePath := filepath.Join(f.config.BaseFilePath, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("crate file failed:%v", err)
		}
		f.file = file
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data filed:%v", err)
	}
	// todo we should close the file
	_, err = f.file.Write(bytes)
	return err
}

func (f *FileOutputer) name() string {
	return f.Name
}

func (f *FileOutputer) createFileName(config *Config, comm string, pid string) string {
	fileName := comm + "_" + pid
	return config.BaseFilePath + fileName
}

type TerminalOutputer struct {
	Name   string
	config *Config
	file   *os.File
}

func (t *TerminalOutputer) output(data *model.DataBlock) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data failed:%v", err)
	}
	fmt.Println("{}", string(bytes))
	return nil
}

func (t *TerminalOutputer) name() string {
	return t.Name
}
