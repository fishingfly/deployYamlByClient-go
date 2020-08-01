package filetype

import (
	"errors"
	"github.com/golang/glog"
	"io/ioutil"
	"strings"
)

type DeployConfig struct {
	Pipeline []Pipeline `json:"pipeline"`
}

type Pipeline struct {
	Name       string   `json:"name"`
	Enable     bool     `json:"enable"`
	External   External `json:"external"`
	Properties []Property `json:"properties"`
}

type External struct {
	ExternalType string `json:"type"`
	Gomod        string `json:"gomod"`
	Subpath      bool `json:"subpath"`
}

type Property struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type taskResult struct {
	Name string `json:"name"`
	Status string `json:"status"`
	Console string `json:"console"`
}

type Result struct {
	Report []taskResult `json:"result"`
}

type Change struct {
	Server string `json:"server"`
	Name string `json:"name"`
	ChangResult string `json:"result"`
}

type DeployResult struct {
	Name string `json:"name"`
	Status string `json:"status"`
	Console string `json:"console"`
	Deploystat Deploystat `json:"deploystat"`
}

type Deploystat struct{
	Changes []*Change `json:"changes"`
}

type ResultInterface struct {
	Report []interface{} `json:"report"`
}

func HasFileInConfPath(resourcePath, filename string) error {
	if len(resourcePath) == 0 {
		return errors.New("resouceFilePath is nil")
	}
	dirList, err := ioutil.ReadDir(resourcePath)
	if err != nil { //有错误，后面就不继续了
		glog.Errorf("hasYamlInConfPath failed, error is %s\n", err.Error())
		return err
	}
	for _, v := range dirList {
		if len(filename) == 0 {
			if !(strings.Contains(v.Name(), "yaml") || strings.Contains(v.Name(), "yml")) {// 有一个包含的就可以了
				return errors.New("yaml path has file not in yaml format")
			}
		} else {
			if v.Name() != filename {
				return errors.New("yaml path don't have kubeconfig file or file is not kubeconfig")
			}
		}

	}
	return nil
}
