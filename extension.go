// Package pipefence is a goldmark extension for transforming fenced code blocks to HTML.
//
// This package can be used to bridge the gap between external tools
// like graphviz and pikchr and goldmark.
package pipefence

import (
	"bytes"
	"fmt"
	"log"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// PipeFunc defines how to transform the contents of a given fenced
// code block.
type PipeFunc func([]byte) ([]byte, error)

// Extension is a goldmark extension which pipes annotated fenced code
// block contents through the matching functions.
//
// For example, if PipeFuncs["pikchr"] is set to be a function that
// converts the pikchr graphical description language to SVG, the
// following fenced code block will render as SVG:
//
//	```pikchr
//	box "lolcat"
//	```
type Extension struct {
	PipeFuncs map[string]PipeFunc
}

// Extension extends the provided Goldmark parser with support for
// Pikchr diagrams.
func (e *Extension) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&transformer{ext: e}, 100),
		),
	)
	md.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(&pfRenderer{ext: e}, 100),
		),
	)
}

// transformer transforms eligible fenced code blocks into pfBlock.
//
// The only purpose of this step is so that we can register a renderer
// for that specific pfBlock node kind, rather than for all fenced
// code blocks.
type transformer struct {
	ext *Extension
}

func (t *transformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	var fencedBlocks []*ast.FencedCodeBlock

	err := ast.Walk(doc, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		fb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}
		fencedBlocks = append(fencedBlocks, fb)
		return ast.WalkContinue, nil
	})
	if err != nil {
		// Can not happen if the AST walking callback does not return errors.
		log.Fatalf("Implementation error: ast.Walk: %v", err)
	}

	for _, fb := range fencedBlocks {
		lang := fb.Language(reader.Source())
		_, ok := t.ext.PipeFuncs[string(lang)]
		if !ok {
			continue
		}

		parent := fb.Parent()
		doc.ReplaceChild(parent, fb, &pfBlock{
			FencedCodeBlock: *fb,
		})
	}
}

var pfKind = ast.NewNodeKind("PipefenceBlock")

// pfBlock is a fenced code block whose content needs to be
// transformed.
//
// This is a thin wrapper around ast.FencedCodeBlock
// so that we can register a special renderer for it.
type pfBlock struct {
	ast.FencedCodeBlock
}

func (b *pfBlock) IsRaw() bool        { return true }
func (b *pfBlock) Kind() ast.NodeKind { return pfKind }
func (b *pfBlock) RawContent(src []byte) []byte {
	lines := b.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(src))
	}
	return buf.Bytes()
}

// pfRenderer renders pfBlocks by piping them through one of the
// PipeFuncs.
type pfRenderer struct {
	ext *Extension
}

func (r *pfRenderer) RegisterFuncs(registry renderer.NodeRendererFuncRegisterer) {
	renderFenced := func(w util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		fb := node.(*pfBlock)
		lang := string(fb.Language(src))
		pipeFunc, ok := r.ext.PipeFuncs[lang]
		if !ok {
			return ast.WalkContinue, nil
		}

		if !entering {
			return ast.WalkContinue, nil
		}

		content, err := pipeFunc(fb.RawContent(src))
		if err != nil {
			return ast.WalkStop, fmt.Errorf("fenced block transformer %q: %v", lang, err)
		}
		w.Write(content)
		return ast.WalkSkipChildren, nil
	}
	registry.Register(pfKind, renderFenced)
}
