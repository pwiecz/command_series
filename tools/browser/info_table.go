package main

import "github.com/pwiecz/go-fltk"

type InfoTable struct {
	*fltk.TableRow
}

func NewInfoTable(x, y, w, h int) *InfoTable {
	t := &InfoTable{}
	t.TableRow = fltk.NewTableRow(x, y, w, h)
	return t
}
