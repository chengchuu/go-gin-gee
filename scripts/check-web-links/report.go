package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var supportedReportTypes = []string{
	"AUDIO", "CSS", "FONT", "HTML", "IMAGE", "JAVASCRIPT", "JSON",
	"MANIFEST", "OTHER", "PDF", "SITEMAP", "TEXT", "UNKNOWN", "VIDEO", "XML",
}

type crawlSnapshot struct {
	root       string
	checked    []string
	types      map[string]string
	failures   []failure
	skipped    []skip
	discovered int
	incomplete bool
}

func (s crawlSnapshot) hasFailures() bool {
	return len(s.failures) > 0 || s.incomplete
}

func (s crawlSnapshot) typeCounts() map[string]int {
	counts := make(map[string]int)
	for _, resourceType := range s.types {
		counts[resourceType]++
	}
	return counts
}

func (s crawlSnapshot) passedCount() int {
	failedChecked := make(map[string]struct{})
	for _, f := range s.failures {
		if _, ok := s.types[stripFragment(f.url)]; ok {
			failedChecked[stripFragment(f.url)] = struct{}{}
		}
	}
	passed := len(s.checked) - len(failedChecked)
	if passed < 0 {
		return 0
	}
	return passed
}

func parseReportTypes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	supported := make(map[string]struct{}, len(supportedReportTypes))
	for _, resourceType := range supportedReportTypes {
		supported[resourceType] = struct{}{}
	}
	selected := make(map[string]struct{})
	var unknown []string
	for _, value := range strings.Split(raw, ",") {
		resourceType := strings.ToUpper(strings.TrimSpace(value))
		if resourceType == "" {
			continue
		}
		if _, ok := supported[resourceType]; !ok {
			unknown = append(unknown, resourceType)
			continue
		}
		selected[resourceType] = struct{}{}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown report type(s): %s; supported types: %s", strings.Join(uniqueStrings(unknown), ", "), strings.Join(supportedReportTypes, ", "))
	}
	return sortedKeys(selected), nil
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	unique := values[:1]
	for _, value := range values[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique
}

func reportTypeSet(selected []string) map[string]struct{} {
	if len(selected) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(selected))
	for _, resourceType := range selected {
		set[resourceType] = struct{}{}
	}
	return set
}

func writeFailures(w io.Writer, failures []failure) {
	if len(failures) == 0 {
		return
	}
	fmt.Fprintln(w, "Failed URLs:")
	for _, f := range failures {
		fmt.Fprintf(w, "  %s -- %s (referred by %s)\n", f.url, f.reason, f.referrer)
	}
}

func writeSkipped(w io.Writer, skipped []skip) {
	fmt.Fprintln(w, "Skipped URLs:")
	for _, s := range skipped {
		fmt.Fprintf(w, "  %s -- %s (referred by %s)\n", s.url, s.reason, s.referrer)
	}
}

func typeCountsLine(snapshot crawlSnapshot) string {
	counts := snapshot.typeCounts()
	parts := make([]string, 0, len(counts))
	for _, resourceType := range sortedKeys(counts) {
		parts = append(parts, fmt.Sprintf("%s=%d", resourceType, counts[resourceType]))
	}
	return "Resource types: " + strings.Join(parts, " ")
}

func summaryLine(snapshot crawlSnapshot) string {
	return fmt.Sprintf("Summary: checked=%d passed=%d failed=%d skipped=%d discovered=%d", len(snapshot.checked), snapshot.passedCount(), len(snapshot.failures), len(snapshot.skipped), snapshot.discovered)
}

func reportConsole(w io.Writer, snapshot crawlSnapshot, showSkipped bool) {
	writeFailures(w, snapshot.failures)
	if showSkipped {
		writeSkipped(w, snapshot.skipped)
	}
	fmt.Fprintln(w, typeCountsLine(snapshot))
	fmt.Fprintln(w, summaryLine(snapshot))
}

func renderFileReport(snapshot crawlSnapshot, opts options) []byte {
	var report bytes.Buffer
	fmt.Fprintln(&report, "Scoped Web Link Report")
	fmt.Fprintf(&report, "Project root: %s\n", snapshot.root)
	if len(opts.reportTypes) == 0 {
		fmt.Fprintln(&report, "Report types: ALL")
	} else {
		fmt.Fprintf(&report, "Report types: %s\n", strings.Join(opts.reportTypes, ","))
	}
	fmt.Fprintln(&report)
	fmt.Fprintln(&report, "Checked URLs:")

	selected := reportTypeSet(opts.reportTypes)
	grouped := make(map[string][]string)
	for _, raw := range snapshot.checked {
		resourceType := snapshot.types[raw]
		if selected != nil {
			if _, ok := selected[resourceType]; !ok {
				continue
			}
		}
		grouped[resourceType] = append(grouped[resourceType], raw)
	}
	for _, resourceType := range sortedKeys(grouped) {
		urls := grouped[resourceType]
		sort.Strings(urls)
		fmt.Fprintf(&report, "  %s (%d):\n", resourceType, len(urls))
		for _, raw := range urls {
			fmt.Fprintf(&report, "    %s\n", raw)
		}
	}

	if len(snapshot.failures) > 0 {
		fmt.Fprintln(&report)
		writeFailures(&report, snapshot.failures)
	}
	if opts.showSkipped {
		fmt.Fprintln(&report)
		writeSkipped(&report, snapshot.skipped)
	}
	fmt.Fprintln(&report)
	fmt.Fprintln(&report, typeCountsLine(snapshot))
	fmt.Fprintln(&report, summaryLine(snapshot))
	return report.Bytes()
}

func writeReportAtomic(filename string, content []byte) error {
	return writeReportAtomicWithRename(filename, content, replaceReportFile)
}

func writeReportAtomicWithRename(filename string, content []byte, rename func(string, string) error) (err error) {
	dir := filepath.Dir(filename)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	closed := false
	defer func() {
		if !closed {
			if closeErr := temp.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
		}
		if removeErr := os.Remove(tempName); err == nil && removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = removeErr
		}
	}()
	if err = temp.Chmod(0o644); err != nil {
		return err
	}
	if _, err = temp.Write(content); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	closed = true
	if err = rename(tempName, filename); err != nil {
		return err
	}
	return nil
}
