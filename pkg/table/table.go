package table

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/olekukonko/tablewriter"
)

type Table struct {
	Header      []string
	Lines       [][]string
	SortIndices []int
	Output      string
}

func (t Table) Len() int      { return len(t.Lines) }
func (t Table) Swap(i, j int) { t.Lines[i], t.Lines[j] = t.Lines[j], t.Lines[i] }
func (t Table) Less(i, j int) bool {
	for _, index := range t.SortIndices {
		if t.Lines[i][index] == t.Lines[j][index] {
			continue
		}
		return compare(t.Lines[i][index],t.Lines[j][index])
	}
	return compare(t.Lines[i][0],t.Lines[j][0])
}

func compare(s1, s2 string) bool {
	s1Time, s1Err := time.Parse("02-01-2006 15:04:05", s1)
	s2Time, s2Err := time.Parse("02-01-2006 15:04:05", s2)
	if s1Err != nil  || s2Err != nil {
		return s1 < s2
	}
	return s1Time.Before(s2Time)
}

func ConvertToTable(t Table) (string, error) {

	buff := &bytes.Buffer{}
	sort.Sort(t)

	switch t.Output {
	case "raw":
		w := tabwriter.NewWriter(buff, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%s\n", strings.Join(t.Header, "\t"))
		for _, line := range t.Lines {
			fmt.Fprintf(w, "%s\n", strings.Join(line, "\t"))
		}
		w.Flush()
	case "markdown":
		table := tablewriter.NewWriter(buff)
		table.SetHeader(t.Header)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		table.SetReflowDuringAutoWrap(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.AppendBulk(t.Lines)
		table.Render()
	default:
		panic(fmt.Errorf("unknown output: %s", t.Output))
	}

	return buff.String(), nil
}
