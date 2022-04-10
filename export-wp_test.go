package main // _test

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
)

var (
	testXML string
)

func TestMain(m *testing.M) {
	fileReader, err := os.OpenFile("testWpExport.xml", os.O_RDONLY, os.ModeCharDevice)
	if err != nil {
		log.Fatalf("could not read test payload: %v", err)
	}
	bts, err := ioutil.ReadAll(fileReader)
	if err != nil {
		log.Fatalf("could not read xml: %v", err)
	}
	testXML = string(bts)
	err = fileReader.Close()
	if err != nil {
		log.Fatalf("could not read xml: %v", err)
	}
	os.Exit(m.Run())
}

func Test_parseXML(t *testing.T) {

	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	if doc.Items[0].Encodeds[0].Data != "http://plazamoyua.files.wordpress.com/2007/01/moyua6.jpg" {
		t.Errorf("unexpected content for item1: %s", doc.Items[0].Encodeds[0].Data)
	}
	if doc.Items[0].Author != "soil" {
		t.Errorf("unexpected author for item1: %s", doc.Items[0].Author)
	}
	if doc.Items[0].Title != "moyua6.jpg" {
		t.Errorf("unexpected title for item1: %s", doc.Items[0].Title)
	}
	if doc.Items[0].PubDate != "Sat, 20 Jan 2007 06:36:31 +0000" {
		t.Errorf("unexpected PubDate for item1: %s", doc.Items[0].PubDate)
	}
	if doc.Items[0].PostType != "attachment" {
		t.Errorf("unexpected PubDate for item1: %s", doc.Items[0].PubDate)
	}
}

func Test_cleanContent(t *testing.T) {
	// http://www.europapress.es/madrid/noticia-ciudadanos-cerca-acuerdo-valia-merino-encabece-lista-ayuntamiento-cerrara-mikel-buesa-20110307173526.html
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	renderer := contentRenderer{
		transformContent: substituteMediaRoot,
	}

	var buff bytes.Buffer
	err = renderer.toMarkdown(doc.Items[0], &buff)
	if err != nil {
		t.Errorf("could not convert post to markdown: %v", err)
	}
	md := buff.String()
	if strings.Contains(md, "plazamoyua.files.wordpress.com") {
		t.Errorf("should have eliminated media url reference")
	}
	if !strings.Contains(md, "http://localhost:1313/media/2007/01/moyua6.jpg") {
		t.Errorf("should have eliminated media url reference")
	}
	t.Log(md)
}

func Test_generateMD(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	renderer := contentRenderer{
		transformContent: func(in string) string { return in },
	}

	var buff bytes.Buffer
	err = renderer.toMarkdown(doc.Items[2], &buff)
	if err != nil {
		t.Errorf("could not convert post to markdown: %v", err)
	}

	md := buff.String()
	expectedWords := []string{"---", "date:", "title:", "author:", "slug:", doc.Items[2].Encodeds[0].Data}
	for _, word := range expectedWords {
		t.Run("testing-for-"+word, func(t *testing.T) {
			if !strings.Contains(md, word) {
				t.Errorf("missing %s from markdown", word)
			}
		})
	}
}

func Test_generateHTMLinMD(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	renderer := contentRenderer{
		transformContent: func(in string) string { return in },
	}

	var buff bytes.Buffer
	err = renderer.toMarkdown(doc.Items[3], &buff)
	if err != nil {
		t.Errorf("could not convert post to markdown: %v", err)
	}

	md := buff.String()

	extracts := []string{
		`title: "Las plataformas de hielo de la Antártida, estables. Lástima por los \"fans\" de Wilkins."`,
		`author: "plazaeme"`,
		`slug: "las-plataformas-de-hielo-de-la-antartida-estables-lo-siento-por-fans-de-wilkins"`,
		`categories: ["algoreros", "calentamiento-global", "cambio-climatico"]`,
		`Científicos de la <em><strong>Western Australia's Curtin University of Technology</strong></em> están usando sensores acústicos,`,
	}

	for _, extract := range extracts {
		if !strings.Contains(md, extract) {
			t.Errorf("missing content: %s", extract)
		}
	}

	t.Log(md)
}

func Test_threadComments(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	comments := doc.Items[2].Comments
	if 3 != len(comments) {
		t.Errorf("unexpected number of comments: %d. Wanted 3", len(comments))
	}

	commentThreads := threadComments(comments)
	if 1 != len(commentThreads) {
		t.Errorf("expected 1 thread, got %d", len(commentThreads))
		t.FailNow()
	}

	if commentThreads[0].ID != commentThreads[0].Children[0].ParentID ||
		commentThreads[0].Children[0].ID != commentThreads[0].Children[0].Children[0].ParentID {
		t.Errorf("unexpected thread %v", commentThreads[0])
	}
}

func Test_renderComments(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Items) != 4 {
		t.Fatal("expected to find 4 items")
	}

	comments := doc.Items[2].Comments
	if 3 != len(comments) {
		t.Fatal("expected 3 comments")
	}

	commentThreads := threadComments(comments)

	renderer := contentRenderer{
		transformContent: func(in string) string { return in },
	}

	html, err := renderer.threadToHTML(commentThreads[0])
	if err != nil {
		t.Fatal(err)
	}

	expectedFragments := []string{
		`<div class="comment">`,
		`<span class="author">MOY</span>`,
		`paco <i>pecho</i> chico rico`,
		`insultaba como loco`,
		`<span class="author">Foo</span>`,
	}

	for _, frag := range expectedFragments {
		if !strings.Contains(string(html), frag) {
			t.Errorf("expected to find string %s", frag)
		}
	}
}

func Test_parseCategoriesTags(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 4 {
		t.Errorf("expected to find 4 items, found %d", len(doc.Items))
		t.FailNow()
	}

	if len(doc.Items[3].Categories) != 4 {
		t.Errorf("expected 4 categories, found %d", len(doc.Items[3].Categories))
	}

	var tags, cats int
	for _, ct := range doc.Items[3].Categories {
		switch ct.Domain {
		case "category":
			cats++
		case "post_tag":
			tags++
		default:
			t.Errorf("unknown domain: %s", ct.Domain)
		}
	}
	if tags != 1 {
		t.Errorf("expected 1 tag, found %d", tags)
	}
	if cats != 3 {
		t.Errorf("expected 3 categories, found %d", tags)
	}
	expected := category{
		XMLName:  xml.Name{Space: "", Local: "category"},
		Domain:   "post_tag",
		NiceName: "cambio-climatico",
		Data:     "Cambio Climático",
	}
	if expected != doc.Items[3].Categories[3] {
		t.Errorf("unexpected tag: %#v", doc.Items[3].Categories[3])
	}
}

func TestLinkify(t *testing.T) {
	in := `Y luego pasó a ser esta dirección. Hace poco funcionaba:
	http://reode.blogspot.com/ y http://spreadsheets.google.com/pub?key=tl4jqSPuZ3CE5wxyDVwjvMA&single=true&gid=0&output=html también,
	Pero ya no <a href="http://amazon.com">amazon</a>.
	Creo que era:

http://ciudadanos-nada-mas.blogspot.com/
Ahora se pueden ver los resultados provisionales con gran comodidad en una hoja de cálculo que ha preparado "Producciones Críticas":

http://spreadsheets.google.com/pub?key=tl4jqSPuZ3CE5wxyDVwjvMA&amp;single=true&amp;gid=0&amp;output=html`

	out, err := linkifyText(in)
	if err != nil {
		t.Fatalf("could not linkify: %v", err)
	}
	frags := []string{
		`<a href="http://reode.blogspot.com/">http://reode.blogspot.com/</a> y `,
		`Pero ya no <a href="http://amazon.com">amazon</a>.`,
		`<a href="http://ciudadanos-nada-mas.blogspot.com/">http://ciudadanos-nada-mas.blogspot.com/</a>`,
		`<a href="http://spreadsheets.google.com/pub?key=tl4jqSPuZ3CE5wxyDVwjvMA&amp;single=true&amp;gid=0&amp;output=html">http://spreadsheets.google.com/pub?key=tl4jqSPuZ3CE5wxyDVwjvMA&amp;single=true&amp;gid=0&amp;output=html</a>`,
	}
	for _, frag := range frags {
		if !strings.Contains(out, frag) {
			t.Errorf("expected to find string %s", frag)
		}
	}
	t.Log(string(out))
}
