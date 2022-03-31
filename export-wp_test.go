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

	if len(doc.Items) != 3 {
		t.Errorf("expected to find 3 items, found %d", len(doc.Items))
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

func Test_generateMD(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 3 {
		t.Errorf("expected to find 3 items, found %d", len(doc.Items))
		t.FailNow()
	}

	var buff bytes.Buffer
	err = doc.Items[2].toMarkdown(&buff)
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

func Test_threadComments(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	if len(doc.Items) != 3 {
		t.Errorf("expected to find 3 items, found %d", len(doc.Items))
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

	if commentThreads[0].Id != commentThreads[0].Children[0].ParentId ||
		commentThreads[0].Children[0].Id != commentThreads[0].Children[0].Children[0].ParentId {
		t.Errorf("unexpected thread %v", commentThreads[0])
	}
}

func Test_renderComments(t *testing.T) {
	var doc rss
	err := xml.Unmarshal([]byte(testXML), &doc)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Items) != 3 {
		t.Fatal("expected to find 3 items")
	}

	comments := doc.Items[2].Comments
	if 3 != len(comments) {
		t.Fatal("expected 3 comments")
	}

	commentThreads := threadComments(comments)

	html, err := threadToHTML(commentThreads[0])
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
