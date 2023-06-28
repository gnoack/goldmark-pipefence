package pipefence_test

import (
	"bytes"
	"testing"

	pipefence "github.com/gnoack/goldmark-pipefence"
	"github.com/yuin/goldmark"
)

func TestPipefence(t *testing.T) {
	gmark := goldmark.New(goldmark.WithExtensions(&pipefence.Extension{
		PipeFuncs: map[string]pipefence.PipeFunc{
			"banana": func(a []byte) ([]byte, error) {
				return bytes.ReplaceAll(a, []byte("o"), []byte("a")), nil
			},
		},
	}))

	for _, tt := range []struct {
		Name  string
		Input string
		Want  string
	}{
		{
			Name:  "Basic",
			Input: "lolcat",
			Want:  "<p>lolcat</p>\n",
		},
		{
			Name:  "RenderFooAsFaa",
			Input: "```banana\nfoo\n```\n",
			Want:  "faa\n",
		},
		{
			Name:  "UnregisteredTransformerRendersLikeNormalFencedBlock",
			Input: "```unknown\nfoo\n```\n",
			Want:  "<pre><code class=\"language-unknown\">foo\n</code></pre>\n",
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			var buf bytes.Buffer
			err := gmark.Convert([]byte(tt.Input), &buf)
			if err != nil {
				t.Fatalf("gmark.Convert: %v", err)
			}

			got := string(buf.Bytes())
			if got != tt.Want {
				t.Errorf("gmark.Convert(%q) = %q, want %q", tt.Input, got, tt.Want)
			}
		})
	}
}
