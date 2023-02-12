package item

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type ID struct {
	Owner   string
	Service string
	Name    string
}

const (
	IDMapsCount = 128
)

type Assignment struct {
	Id     ID
	Value  string
	Expiry *time.Time
}

var parseID *regexp.Regexp
var parseAssn *regexp.Regexp

func init() {
	parseID = regexp.MustCompile(`(?P<ow>[^:]*)\:(?P<se>[^:]*)\:(?P<na>.*)`)
	parseAssn = regexp.MustCompile(`(?P<ow>[^:]*)\:(?P<se>[^:]*)\:(?P<na>[^=]*)\=(?P<va>[^,]*)(\,(?P<ex>[0-9]+))?`)
}

func (id *ID) HashKey() int {
	hash := 0
	for _, char := range id.Owner {
		hash = (hash ^ int(char)) % IDMapsCount
	}
	for _, char := range id.Service {
		hash = (hash ^ int(char)) % IDMapsCount
	}
	for _, char := range id.Name {
		hash = (hash ^ int(char)) % IDMapsCount
	}
	return hash
}

func (id *ID) Compose() string {
	return fmt.Sprintf("%s:%s:%s", id.Owner, id.Service, id.Name)
}

func (id *ID) Parse(s string) error {
	match := parseID.FindStringSubmatch(s)
	for i, name := range parseID.SubexpNames() {
		if i > 0 && i <= len(match) {
			switch name {
			case "ow":
				id.Owner = match[i]
			case "se":
				id.Service = match[i]
			case "na":
				id.Name = match[i]
			}
		}
	}
	if id.Service == "" || id.Name == "" {
		return errors.New("incorrect assignment " + s)
	}
	return nil
}

func ParseAssignment(s string) (Assignment, error) {
	match := parseAssn.FindStringSubmatch(s)
	as := Assignment{}
	for i, name := range parseAssn.SubexpNames() {
		if i > 0 && i <= len(match) {
			switch name {
			case "ow":
				as.Id.Owner = match[i]
			case "se":
				as.Id.Service = match[i]
			case "na":
				as.Id.Name = match[i]
			case "va":
				as.Value = match[i]
			case "ex":
				if match[i] != "" {
					timeNow := time.Now()
					as.Expiry = &timeNow
					expSeconds, _ := strconv.Atoi(match[i])
					*as.Expiry = as.Expiry.Add(time.Second * time.Duration(expSeconds))
				} else {
					as.Expiry = nil
				}
			}
		}
	}
	if as.Id.Service == "" || as.Id.Name == "" {
		return Assignment{}, errors.New("incorrect assignment " + s)
	}
	return as, nil
}
