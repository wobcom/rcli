package api

import (
	"encoding/xml"
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/gookit/slog"
	"os"
	"regexp"
	"strings"
)

func ParseDiffFromText(text string) (*JunosDiff, error) {

	parseDummy := JunosConfInfo{}

	err := xml.Unmarshal([]byte(text), &parseDummy)
	if err != nil {
		return nil, err
	}

	junosConf := parseDummy.ConfigurationOutput
	junosConf.ConfType = ConftypeText
	junosConf.Diff = strings.TrimSpace(junosConf.Diff)
	junosConf.IsEmpty = junosConf.Diff == ""

	return &junosConf, nil
}

type JunosDiff struct {
	ConfType ConfType `xml:"-"`
	Diff     string   `xml:",innerxml"`
	IsEmpty  bool     `xml:"-"`
	XMLName  struct{} `xml:"configuration-output"`
}

type JunosConfInfo struct {
	XMLName             struct{}  `xml:"configuration-information"`
	ConfigurationOutput JunosDiff `xml:"configuration-output"`
}

func (jD *JunosDiff) WriteToFile(filePath string) error {
	slog.Info(fmt.Sprintf("Writing diff to %v", filePath))
	return os.WriteFile(filePath, []byte(jD.Diff), 0644)
}

func (jD *JunosDiff) Print() {
	lines := strings.Split(jD.Diff, "\n")

	diffRegexp := regexp.MustCompile(`^\[([\w-. ]*)\]$`)

	isSkipping := false
	for _, line := range lines {

		diffMatch := diffRegexp.FindStringSubmatch(line)

		if len(diffMatch) >= 2 {
			// This is a new diff line. We may want to reset some state here and check, if this should be omitted or not.
			isSkipping = false

			skippedStarts := []string{
				"edit policy-options as-path-group",
				"edit policy-options prefix-list",
			}

			hasSomeMatch := false
			for _, startPrefix := range skippedStarts {
				if strings.HasPrefix(diffMatch[1], startPrefix) {
					hasSomeMatch = true
				}
			}

			if hasSomeMatch {
				isSkipping = true
				fmt.Println(color.InYellow(fmt.Sprintf("[omitting %v]", diffMatch[1])))
			} else {
				fmt.Println(color.InPurple(line))
			}
		} else if isSkipping {
			continue
		} else if strings.HasPrefix(line, "+") {
			fmt.Println(color.InGreen(line))
		} else if strings.HasPrefix(line, "-") {
			fmt.Println(color.InRed(line))
		} else {
			fmt.Println(line)
		}
	}
}
