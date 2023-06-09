package api

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ConfType does track the type of the config.
// Junos supports different configuration formats and we will support all of them at some time.
type ConfType string

const (
	ConftypeText ConfType = "text"
)

type JunosConfiguration struct {
	ConfType ConfType `xml:"-"`
	Text     string   `xml:",innerxml"`
	XMLName  struct{} `xml:"configuration-text"`
}

func ParseFromText(text string) (*JunosConfiguration, error) {

	junosConf := JunosConfiguration{
		ConfType: ConftypeText,
	}

	err := xml.Unmarshal([]byte(text), &junosConf)
	if err != nil {
		return nil, err
	}

	return &junosConf, nil
}

func ParseFromFile(filePath string, version string) (*JunosConfiguration, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	versionString := fmt.Sprintf("version %v;\n", version)

	s := versionString + string(fileData)

	// Technically, this could contain other urlencoded characters, but Junos does not use those.
	s = strings.Replace(s, "<", "&lt;", -1)
	s = strings.Replace(s, ">", "&gt;", -1)
	junosConf := JunosConfiguration{
		ConfType: ConftypeText,
		Text:     s,
	}
	return &junosConf, nil
}

func (j *JunosConfiguration) ToText() string {
	b, _ := xml.Marshal(j)
	return string(b)
}

type RPCErrorInfo struct {
	XMLName    struct{} `xml:"error-info"`
	BadElement string   `xml:"bad-element"`
}

type RPCError struct {
	XMLName       struct{}     `xml:"rpc-error"`
	ErrorSeverity string       `xml:"error-severity"`
	ErrorPath     string       `xml:"error-path"`
	ErrorMessage  string       `xml:"error-message"`
	ErrorInfo     RPCErrorInfo `xml:"error-info"`
}

type JunosLoadConfigurationResults struct {
	XMLName  struct{}  `xml:"load-configuration-results"`
	Ok       *struct{} `xml:"ok"`
	RpcError RPCError  `xml:"rpc-error"`
}

func ParseLoadConfigurationResultsFromText(text string) error {

	parseDummy := JunosLoadConfigurationResults{}

	err := xml.Unmarshal([]byte(text), &parseDummy)
	if err != nil {
		return err
	}

	if parseDummy.Ok != nil {
		return nil
	}

	return errors.New(fmt.Sprintf("RPC error %v: Bad Element %v", parseDummy.RpcError.ErrorMessage, parseDummy.RpcError.ErrorInfo.BadElement))
}

type JunosCommandResults struct {
	Output  string   `xml:",innerxml"`
	XMLName struct{} `xml:"output"`
}

func ParseCommandResultsFromText(text string, format string) (string, error) {
	commandResult := JunosCommandResults{}

	if format != "text" {
		return text, nil
	}

	err := xml.Unmarshal([]byte(text), &commandResult)
	if err != nil {
		return "", err
	}

	return commandResult.Output, nil
}

type JunosVersion struct {
	SoftwareInformation []struct {
		HostName []struct {
			Data string `json:"data"`
		} `json:"host-name"`
		ProductModel []struct {
			Data string `json:"data"`
		} `json:"product-model"`
		ProductName []struct {
			Data string `json:"data"`
		} `json:"product-name"`
		JunosVersion []struct {
			Data string `json:"data"`
		} `json:"junos-version"`
	} `json:"software-information"`
}

func ParseJunosVersionFromJson(text string) (string, error) {

	jVersion := JunosVersion{}

	err := json.Unmarshal([]byte(text), &jVersion)
	if err != nil {
		return "", err
	}

	return jVersion.SoftwareInformation[0].JunosVersion[0].Data, nil
}
