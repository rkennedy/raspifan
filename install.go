package main

import (
	_ "embed"
	"os"
	"text/template"
)

//go:embed raspifan.service.template
var serviceTemplate string

type ServiceParameters struct {
	TempPath string
	SelfPath string
}

func install() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	destination, err := os.Open("/usr/local/lib/systemd/system/raspifan.service")
	if err != nil {
		return err
	}
	defer destination.Close()

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return err
	}
	params := ServiceParameters{
		TempPath: temperaturePath,
		SelfPath: self,
	}
	return tmpl.Execute(destination, params)
}
