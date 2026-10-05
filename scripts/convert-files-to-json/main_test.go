package main

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

func TestFileListingAndProjectFilter(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("dist", 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"demo.template.html", "demo.degrade.html", "demo.client.js", "demo.server.js", "unrelated.server.js"} {
		if err := os.WriteFile("dist/"+name, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	originalFlags, originalArgs := flag.CommandLine, os.Args
	t.Cleanup(func() { flag.CommandLine, os.Args = originalFlags, originalArgs })
	flag.CommandLine = flag.NewFlagSet("fixture", flag.ContinueOnError)
	os.Args = []string{"fixture", "-path=./dist", "-projects=demo", "-baseurl=https://example.test/", "-outfile=result.json"}
	main()
	data, err := os.ReadFile("result.json")
	if err != nil {
		t.Fatal(err)
	}
	var got GamecenterH5
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := GamecenterH5{Bundleurl: "https://example.test/demo.server.js", Degradeurl: "https://example.test/demo.degrade.html", Viewurl: "https://example.test/demo.template.html", Manifesturl: "https://example.test/demo.client.js"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
