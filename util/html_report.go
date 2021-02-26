package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	MarkPass    = " ------ " + `<span class="material-icons" style="position: relative; top: 0.3em; color: green; font-weight: bold;">done</span>` + "\n"
	MarkFail    = " ------ " + `<span class="material-icons" style="position: relative; top: 0.3em; color: red; font-weight: bold;">clear</span>` + "\n"
	MarkPending = " ------ " + `<span class="material-icons" style="position: relative; top: 0.3em; color: red; font-weight: bold;">sync</span>` + "\n"
)

type TblHeader struct {
	Key    string
	Header string
}
type TblHLColumn struct {
	Mark   int
	Header string
}
type TblHLRow struct {
	Mark  int
	Cells []interface{}
}
type TableWithHighlight struct {
	Column []TblHLColumn
	Row    []TblHLRow
}
type TableBuilder struct {
	Data       []interface{}
	Headers    []TblHeader
	ColHLs     []string // header key of the highlight column
	RowHLs     map[string]interface{}
	FullBorder bool
}

func (d *TableBuilder) SetHeaders(h ...[]string) {
	d.Headers = []TblHeader{}
	if len(h) < 1 || len(h) > 2 {
		return
	}
	for i, th := range h[0] {
		head := TblHeader{th, strings.Title(th)} // default Header is title case of the key
		if len(h) == 2 {
			head.Header = h[1][i] // customized header
		}
		d.Headers = append(d.Headers, head)
	}
}

func (d *TableBuilder) Build() string {
	tb := TableWithHighlight{[]TblHLColumn{}, []TblHLRow{}}
	for _, h := range d.Headers {
		th := TblHLColumn{0, h.Header}
		if InStrings(h.Key, d.ColHLs) {
			th.Mark = 10
		}
		tb.Column = append(tb.Column, th)
	}
	for _, v := range d.Data {
		var vm map[string]interface{}
		if structs.IsStruct(v) {
			//vm = structs.Map(v)
			for _, field := range structs.Fields(v) {
				if structs.IsStruct(field.Value()) {
					if s, ok := field.Value().(interface{ String() string }); ok {
						vm[field.Name()] = s.String()
					} else if fd, ok := field.FieldOk("Name"); ok {
						vm[field.Name()] = fd.Value()
					} else {
						vm[field.Name()] = structs.Map(field.Value())
					}
				} else {
					vm[field.Name()] = field.Value()
				}
			}
		} else {
			if vmr, ok := v.(map[string]interface{}); ok {
				vm = vmr
			} else {
				continue
			}
		}
		tr := TblHLRow{0, []interface{}{}}
		for _, h := range d.Headers {
			tr.Cells = append(tr.Cells, vm[h.Key])
		}
		f := true
		for kh, vh := range d.RowHLs {
			if reflect.DeepEqual(vm[kh], vh) {
				f = f && true
			} else {
				f = f && false
			}
		}
		if f && len(d.RowHLs) > 0 {
			tr.Mark = 10
		}
		tb.Row = append(tb.Row, tr)
	}
	if len(d.RowHLs) == 0 {
		return tb.MakeHtmlTable(true, d.FullBorder)
	}
	return tb.MakeHtmlTable(false, d.FullBorder)
}

func (d *TableWithHighlight) MakeHtmlTable(AltRow, fullBorder bool) string {
	contains := func(p []int, t int) bool {
		// check if the slice contains the number
		for _, v := range p {
			if t == v {
				return true
			}
		}
		return false
	}

	bdr := ` style="border-bottom: 1px solid #ddd; `
	table := `<div style="overflow-x: auto;"><table><tr>`
	if fullBorder {
		bdr = ` style="border: 1px solid #dddddd; `
		table = `<div style="overflow-x: auto;"><table style="border-collapse: collapse;"><tr>`
	}
	styleBase := bdr + `text-align: left; padding: 8px;">`
	styleHL := bdr + `text-align: left; padding: 8px; font-weight: bold; text-shadow: 3px 3px 2px black;">`
	styleSL := bdr + `text-align: left; padding: 8px; background-color: #54a348">`
	styleHLSL := bdr + `text-align: left; padding: 8px; font-weight: bold; text-shadow: 3px 3px 2px black; background-color: #54a348">`
	tre := `<tr style="background-color: #646464;">`
	var hcol []int
	for i, col := range d.Column {
		if col.Mark != 0 {
			hcol = append(hcol, i)
			table += `<th` + styleHL + col.Header + `</th>`
		} else {
			table += `<th` + styleBase + col.Header + `</th>`
		}
	}
	table += `</tr>`
	bgF := true

	for _, row := range d.Row {
		if bgF && AltRow {
			// even row, grey background
			table += tre
		} else {
			table += `<tr>`
		}
		bgF = !bgF
		for i, cell := range row.Cells {
			cellStr := fmt.Sprintf("%v", cell) + `</td>`
			if row.Mark != 0 {
				if contains(hcol, i) {
					table += `<td` + styleHLSL + cellStr
				} else {
					table += `<td` + styleSL + cellStr
				}
			} else if contains(hcol, i) {
				table += `<td` + styleHL + cellStr
			} else {
				table += `<td` + styleBase + cellStr
			}
		}
		table += `</tr>`
	}
	return table + "</table></div>"
}

// MakeHtmlTable convert [][]string to html table (dark scene), auto scroll x
func MakeHtmlTable(src [][]string) string {
	th := `<th style="border: 1px solid #dddddd; text-align: left; padding: 8px;">`
	td := `<td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">`
	tre := `<tr style="background-color: #646464;">`
	// header row
	table := `<div style="overflow-x: auto;"><table style="border-collapse: collapse;"><tr>`
	for _, columnHead := range src[0] {
		table += th + columnHead + `</th>`
	}
	table += `</tr>`
	bgF := true

	for _, row := range src[1:] {
		if bgF {
			// even row, grey background
			table += tre
		} else {
			table += `<tr>`
		}
		bgF = !bgF
		for _, cell := range row {
			table += td + cell + `</td>`
		}
		table += `</tr>`
	}
	return table + "</table></div>"
}

// DiffTxtInPretty is a modified DiffPrettyHtml function
// apply html escape before convert
// generate html code to be used within in <pre>
// optimized for dark background
func DiffTxtInPretty(dmp *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		// text := strings.Replace(html.EscapeString(diff.Text), "\n", "&para;<br>", -1)
		text := html.EscapeString(diff.Text)
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("<ins style=\"background:#00ff00; color:black;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</ins>")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("<del style=\"background:#ff3636; color:black;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</del>")
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString("<span>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		}
	}
	return buff.String()
}

// DiffHtmlInPretty is a modified DiffPrettyHtml function
// no html escape, original []Diff was generated from html block
// generate html code to be used within in <pre>
// optimized for dark background
func DiffHtmlInPretty(dmp *diffmatchpatch.DiffMatchPatch, diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		// text := strings.Replace(html.EscapeString(diff.Text), "\n", "&para;<br>", -1)
		text := diff.Text
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString("<ins style=\"background:#00ff00; color:black;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</ins>")
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString("<del style=\"background:#ff3636; color:black;\">")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</del>")
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString("<span>")
			_, _ = buff.WriteString(text)
			_, _ = buff.WriteString("</span>")
		}
	}
	return buff.String()
}

// GetStructDiff pretty jsonfy two structs and return the html diff result
func GetStructDiff(s1, s2 interface{}) string {
	ss1, _ := json.MarshalIndent(s1, "", "    ")
	ss2, _ := json.MarshalIndent(s2, "", "    ")
	diff := diffmatchpatch.New()
	sdf := diff.DiffCleanupSemanticLossless(diff.DiffMain(string(ss1), string(ss2), true))
	return `<div>` + DiffHtmlInPretty(diff, sdf) + `</div>`
}
