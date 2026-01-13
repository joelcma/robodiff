package robotdiff

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

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
	Ifs      []If      `xml:"if"`
	Fors     []For     `xml:"for"`
	Body     []BodyItem `xml:"-"`
}

type Keyword struct {
	Name      string    `xml:"name,attr"`
	Type      string    `xml:"type,attr"`
	Keywords  []Keyword `xml:"kw"`
	Ifs       []If      `xml:"if"`
	Fors      []For     `xml:"for"`
	Arguments []string  `xml:"arg"`
	Messages  []Message `xml:"msg"`
	Status    Status    `xml:"status"`
	Body      []BodyItem `xml:"-"`
}

// BodyItem represents an ordered child element of a container where Robot may
// interleave <kw>, <if> and <for>. We convert IF/FOR into pseudo-keywords
// later for the UI, but we must preserve order.
type BodyItem struct {
	Keyword *Keyword
	If      *If
	For     *For
}

// Control structures (Robot Framework IF/FOR)
// These appear in output.xml for newer Robot versions.
type If struct {
	Branches []Branch `xml:"branch"`
	Status   Status   `xml:"status"`
}

type Branch struct {
	Type      string   `xml:"type,attr"`
	Condition string   `xml:"condition,attr"`
	Keywords  []Keyword `xml:"kw"`
	Ifs       []If      `xml:"if"`
	Fors      []For     `xml:"for"`
	Return    *Return   `xml:"return"`
	Status    Status    `xml:"status"`
	Body      []BodyItem `xml:"-"`
}

type For struct {
	Flavor string `xml:"flavor,attr"`
	Iter   []Iter `xml:"iter"`
	Var    []string `xml:"var"`
	Value  []string `xml:"value"`
	Status Status `xml:"status"`
}

type Iter struct {
	Keywords []Keyword `xml:"kw"`
	Ifs      []If      `xml:"if"`
	Fors     []For     `xml:"for"`
	Return   *Return   `xml:"return"`
	Status   Status    `xml:"status"`
	Body     []BodyItem `xml:"-"`
}

type Return struct {
	Value  []string `xml:"value"`
	Status Status   `xml:"status"`
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

// --- Order-preserving unmarshalling for mixed bodies (kw/if/for) ---

func (t *Test) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*t = Test{}
	for _, a := range start.Attr {
		if a.Name.Local == "name" {
			t.Name = a.Value
		}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "status":
				var st Status
				if err := d.DecodeElement(&st, &se); err != nil {
					return err
				}
				t.Status = st
			case "kw":
				var kw Keyword
				if err := d.DecodeElement(&kw, &se); err != nil {
					return err
				}
				t.Keywords = append(t.Keywords, kw)
				t.Body = append(t.Body, BodyItem{Keyword: &kw})
			case "if":
				var ifblk If
				if err := d.DecodeElement(&ifblk, &se); err != nil {
					return err
				}
				t.Ifs = append(t.Ifs, ifblk)
				t.Body = append(t.Body, BodyItem{If: &ifblk})
			case "for":
				var forblk For
				if err := d.DecodeElement(&forblk, &se); err != nil {
					return err
				}
				t.Fors = append(t.Fors, forblk)
				t.Body = append(t.Body, BodyItem{For: &forblk})
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if se.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (k *Keyword) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*k = Keyword{}
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "name":
			k.Name = a.Value
		case "type":
			k.Type = a.Value
		}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "kw":
				var child Keyword
				if err := d.DecodeElement(&child, &se); err != nil {
					return err
				}
				k.Keywords = append(k.Keywords, child)
				k.Body = append(k.Body, BodyItem{Keyword: &child})
			case "if":
				var ifblk If
				if err := d.DecodeElement(&ifblk, &se); err != nil {
					return err
				}
				k.Ifs = append(k.Ifs, ifblk)
				k.Body = append(k.Body, BodyItem{If: &ifblk})
			case "for":
				var forblk For
				if err := d.DecodeElement(&forblk, &se); err != nil {
					return err
				}
				k.Fors = append(k.Fors, forblk)
				k.Body = append(k.Body, BodyItem{For: &forblk})
			case "arg":
				var arg string
				if err := d.DecodeElement(&arg, &se); err != nil {
					return err
				}
				k.Arguments = append(k.Arguments, arg)
			case "msg":
				var msg Message
				if err := d.DecodeElement(&msg, &se); err != nil {
					return err
				}
				k.Messages = append(k.Messages, msg)
			case "status":
				var st Status
				if err := d.DecodeElement(&st, &se); err != nil {
					return err
				}
				k.Status = st
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if se.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (b *Branch) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*b = Branch{}
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "type":
			b.Type = a.Value
		case "condition":
			b.Condition = a.Value
		}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "kw":
				var kw Keyword
				if err := d.DecodeElement(&kw, &se); err != nil {
					return err
				}
				b.Keywords = append(b.Keywords, kw)
				b.Body = append(b.Body, BodyItem{Keyword: &kw})
			case "if":
				var ifblk If
				if err := d.DecodeElement(&ifblk, &se); err != nil {
					return err
				}
				b.Ifs = append(b.Ifs, ifblk)
				b.Body = append(b.Body, BodyItem{If: &ifblk})
			case "for":
				var forblk For
				if err := d.DecodeElement(&forblk, &se); err != nil {
					return err
				}
				b.Fors = append(b.Fors, forblk)
				b.Body = append(b.Body, BodyItem{For: &forblk})
			case "return":
				var ret Return
				if err := d.DecodeElement(&ret, &se); err != nil {
					return err
				}
				b.Return = &ret
			case "status":
				var st Status
				if err := d.DecodeElement(&st, &se); err != nil {
					return err
				}
				b.Status = st
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if se.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (it *Iter) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*it = Iter{}
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "kw":
				var kw Keyword
				if err := d.DecodeElement(&kw, &se); err != nil {
					return err
				}
				it.Keywords = append(it.Keywords, kw)
				it.Body = append(it.Body, BodyItem{Keyword: &kw})
			case "if":
				var ifblk If
				if err := d.DecodeElement(&ifblk, &se); err != nil {
					return err
				}
				it.Ifs = append(it.Ifs, ifblk)
				it.Body = append(it.Body, BodyItem{If: &ifblk})
			case "for":
				var forblk For
				if err := d.DecodeElement(&forblk, &se); err != nil {
					return err
				}
				it.Fors = append(it.Fors, forblk)
				it.Body = append(it.Body, BodyItem{For: &forblk})
			case "return":
				var ret Return
				if err := d.DecodeElement(&ret, &se); err != nil {
					return err
				}
				it.Return = &ret
			case "status":
				var st Status
				if err := d.DecodeElement(&st, &se); err != nil {
					return err
				}
				it.Status = st
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if se.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

// --- Compatibility unmarshalling for Robot 7+ attribute names ---

func (m *Message) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*m = Message{}
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "level":
			m.Level = a.Value
		case "timestamp", "time":
			m.Timestamp = a.Value
		}
	}
	var text string
	if err := d.DecodeElement(&text, &start); err != nil {
		return err
	}
	m.Text = text
	return nil
}

func (s *Status) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*s = Status{}
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "status":
			s.Status = a.Value
		case "starttime", "start":
			s.StartTime = a.Value
		case "endtime", "end":
			s.EndTime = a.Value
		case "elapsed":
			// Ignore for now. Some Robot versions provide elapsed instead of end.
			// We keep StartTime and Status which are sufficient for the UI.
		}
	}

	// Drain any nested content if present.
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if end, ok := tok.(xml.EndElement); ok && end.Name.Local == start.Name.Local {
			return nil
		}
	}
}

// sanity helpers used by UI pseudo-kw naming in other packages (kept here for debugging)
func (b BodyItem) String() string {
	switch {
	case b.Keyword != nil:
		return fmt.Sprintf("kw:%s", b.Keyword.Name)
	case b.If != nil:
		return "if"
	case b.For != nil:
		return "for"
	default:
		return "<empty>"
	}
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
