package printers

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/golangci/golangci-lint/pkg/logutils"
	"github.com/golangci/golangci-lint/pkg/result"
)

type Tab struct {
	printLinterName bool
	silent          bool
	log             logutils.Log
}

func NewTab(printLinterName bool, silent bool, log logutils.Log) *Tab {
	return &Tab{
		printLinterName: printLinterName,
		silent:          silent,
		log:             log,
	}
}

func (p Tab) SprintfColored(ca color.Attribute, format string, args ...interface{}) string {
	c := color.New(ca)
	return c.Sprintf(format, args...)
}

func (p *Tab) Print(ctx context.Context, issues <-chan result.Issue) (bool, error) {
	w := tabwriter.NewWriter(logutils.StdOut, 0, 0, 2, ' ', 0)

	issuesN := 0
	for i := range issues {
		issuesN++
		p.printIssue(&i, w)
	}

	if issuesN != 0 {
		p.log.Infof("Found %d issues", issuesN)
	} else if ctx.Err() == nil { // don't print "congrats" if timeouted
		if !p.silent {
			outStr := p.SprintfColored(color.FgGreen, "Congrats! No issues were found.")
			fmt.Fprintln(logutils.StdOut, outStr)
		}
	}

	if err := w.Flush(); err != nil {
		p.log.Warnf("Can't flush tab writer: %s", err)
	}

	return issuesN != 0, nil
}

func (p Tab) printIssue(i *result.Issue, w io.Writer) {
	text := p.SprintfColored(color.FgRed, "%s", i.Text)
	if p.printLinterName {
		text = fmt.Sprintf("%s\t%s", i.FromLinter, text)
	}

	pos := p.SprintfColored(color.Bold, "%s:%d", i.FilePath(), i.Line())
	if i.Pos.Column != 0 {
		pos += fmt.Sprintf(":%d", i.Pos.Column)
	}

	fmt.Fprintf(w, "%s\t%s\n", pos, text)
}
