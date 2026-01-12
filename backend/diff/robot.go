package robotdiff

import "encoding/xml"

// Robot Framework XML structures
type Robot struct {
	XMLName xml.Name `xml:"robot"`
	Suite   Suite    `xml:"suite"`
}

type Suite struct {
	Name   string  `xml:"name,attr"`
	Suites []Suite `xml:"suite"`
	Tests  []Test  `xml:"test"`
	Status Status  `xml:"status"`
}

type Test struct {
	Name     string    `xml:"name,attr"`
	Status   Status    `xml:"status"`
	Keywords []Keyword `xml:"kw"`
}

type Keyword struct {
	Name      string    `xml:"name,attr"`
	Type      string    `xml:"type,attr"`
	Keywords  []Keyword `xml:"kw"`
	Arguments []string  `xml:"arg"`
	Messages  []Message `xml:"msg"`
	Status    Status    `xml:"status"`
}

type Message struct {
	Level     string `xml:"level,attr"`
	Timestamp string `xml:"timestamp,attr"`
	Text      string `xml:",chardata"`
}

type Status struct {
	Status    string `xml:"status,attr"`
	StartTime string `xml:"starttime,attr"`
	EndTime   string `xml:"endtime,attr"`
}
