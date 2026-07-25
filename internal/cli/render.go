// render.go: presentation only - one render function per result type
// (PartitionLayout, InitResult, Plan, …), each taking the io.Writer it
// prints to and returning write errors. Renders whatever data the app
// returned; no decisions, no queries, no I/O beyond the writer.
package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/RTolkachev/horus/internal/domain"
)

// renderLayouts prints one block per table, blank-line separated.
func renderLayouts(w io.Writer, layouts []domain.PartitionLayout) error {
	for i, l := range layouts {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := renderLayout(w, l); err != nil {
			return err
		}
	}
	return nil
}

// renderLayout prints one table: a heading with the watermark and
// snapshot time, then the partitions in bound order.
func renderLayout(w io.Writer, l domain.PartitionLayout) error {
	if !l.Partitioned {
		_, err := fmt.Fprintf(w, "%s: not partitioned\n", l.Table)
		return err
	}
	_, err := fmt.Fprintf(w, "%s - watermark %d, observed %s\n",
		l.Table, l.Watermark, l.TakenAt.Format("2006-01-02 15:04:05 MST"))
	if err != nil {
		return err
	}
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers("PARTITION", "LESS THAN", "~ROWS", "SIZE").
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			if col > 0 {
				s = s.Align(lipgloss.Right)
			}
			if row == table.HeaderRow {
				s = s.Bold(true)
			}
			return s
		})
	for _, p := range l.Partitions {
		bound := strconv.FormatInt(p.UpperBound, 10)
		if p.IsCatchAll {
			bound = "MAXVALUE"
		}
		t.Row(p.Name, bound, strconv.FormatInt(p.ApproxRows, 10), humanBytes(p.Bytes))
	}
	_, err = fmt.Fprintln(w, t.Render())
	return err
}

// humanBytes formats catalog byte counts as binary units (KiB, MiB, …);
// one decimal is plenty for numbers that are estimates to begin with.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
