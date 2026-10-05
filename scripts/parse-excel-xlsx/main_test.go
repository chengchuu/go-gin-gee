package main

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/szyhf/go-excel"
)

// Build a minimal workbook rather than depending on personal spreadsheet data.
func workbookFixture(t *testing.T) string {
	t.Helper()
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	for name, content := range map[string]string{
		"xl/workbook.xml":            `<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="fixture" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/worksheets/sheet1.xml":   `<worksheet><sheetData><row r="1"><c r="A1" t="str"><v>url</v></c><c r="B1" t="str"><v>note</v></c></row><row r="2"><c r="A2" t="str"><v>https://example.test/?exampleName=Alpha&amp;exampleId=7</v></c><c r="B2" t="str"><v>one</v></c></row><row r="3"><c r="A3" t="str"><v>https://example.test/?exampleName=Alpha&amp;exampleId=7</v></c></row></sheetData></worksheet>`,
	} {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.xlsx")
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSpreadsheetReaderContract(t *testing.T) {
	path := workbookFixture(t)
	conn := excel.NewConnecter()
	if err := conn.Open(path); err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.NewReader("missing"); err == nil {
		t.Fatal("missing sheet accepted")
	}
	rd, err := conn.NewReader("fixture")
	if err != nil {
		t.Fatal(err)
	}
	defer rd.Close()
	if got := rd.GetTitles(); !reflect.DeepEqual(got, []string{"url", "note"}) {
		t.Fatalf("titles: %v", got)
	}
	var notes []string
	for rd.Next() {
		var row struct {
			URL  string `xlsx:"url"`
			Note string `xlsx:"note"`
		}
		if err := rd.Read(&row); err != nil {
			t.Fatal(err)
		}
		if row.URL != "https://example.test/?exampleName=Alpha&exampleId=7" {
			t.Fatalf("URL: %q", row.URL)
		}
		notes = append(notes, row.Note)
	}
	if !reflect.DeepEqual(notes, []string{"one", ""}) {
		t.Fatalf("notes: %v", notes)
	}
	var output bytes.Buffer
	original := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(original) })
	defaultUsage(path, "fixture")
	for _, want := range []string{"name: Alpha, count: 2", "id: 7, count: 2"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q in %q", want, output.String())
		}
	}
}

func TestSpreadsheetOpenErrors(t *testing.T) {
	conn := excel.NewConnecter()
	defer conn.Close()
	if err := conn.Open(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil {
		t.Fatal("missing file accepted")
	}
	if err := conn.OpenBinary([]byte("not a workbook")); err == nil {
		t.Fatal("invalid archive accepted")
	}
}
