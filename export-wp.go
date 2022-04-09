package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	textTpl "text/template"
	"time"
)

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []item   `xml:"channel>item"`
}

// item is the place where posts, pages and attachments are represented
type item struct {
	XMLName       xml.Name
	Title         string     `xml:"title"`
	Link          string     `xml:"link"`
	PubDate       string     `xml:"pubDate"`
	Author        string     `xml:"creator"`       // space: dc
	PostDate      string     `xml:"post_date"`     // space: wp
	Slug          string     `xml:"post_name"`     // space: wp
	PostDateGMT   string     `xml:"post_date_gmt"` // space: wp
	Encodeds      []encoded  `xml:"encoded"`       // space: content
	PostMeta      postMeta   `xml:"postmeta"`
	Comments      []comment  `xml:"comment"`
	ID            int        `xml:"post_id"`
	CommentStatus string     `xml:"comment_status"`
	PostParent    int        `xml:"post_parent"`
	PostType      string     `xml:"post_type"`
	Status        string     `xml:"status"` // space: wp
	Categories    []category `xml:"category"`
}

// category represents a category or tag
type category struct {
	XMLName  xml.Name
	Domain   string `xml:"domain,attr"`
	NiceName string `xml:"nicename,attr"`
	Data     string `xml:",cdata"`
}

// encoded represents the payload of an Item - may be content/excerpt
type encoded struct {
	XMLName xml.Name
	Data    string `xml:",cdata"`
}

type postMeta struct {
	XMLName   xml.Name
	MetaKey   string    `xml:"meta_key"`
	MetaValue metaValue `xml:"meta_value"`
}

type metaValue struct {
	XMLName xml.Name
	Value   string `xml:",cdata"`
}

// comment represents a comment on the site, not an XML comment
// the payload `Content` may contain embedded HTML, and is assumed to be
// safe HTML
type comment struct {
	XMLName        xml.Name
	AuthorName     string        `xml:"comment_author"`
	AuthorEmail    string        `xml:"comment_author_email"`
	AuthorURL      string        `xml:"comment_author_url"`
	Content        template.HTML `xml:"comment_content"`
	ID             int           `xml:"comment_id"`
	ParentID       int           `xml:"comment_parent"`
	CommentDate    string        `xml:"comment_date"`
	CommentDateGMT string        `xml:"comment_date_gmt"`
	CreatedAt      time.Time
}

// commentThread represents a comment and its descendents
type commentThread struct {
	comment
	Children     []commentThread
	ChildrenHTML []template.HTML // this is just a convenience for rendering
}

func main() {
	var (
		outdir, xmlFilename string
		localMedia          string // the url for media the WP site served itself
	)

	flag.StringVar(&outdir, "outdir", "", "name of the output directory")
	flag.StringVar(&xmlFilename, "xmlfile", "", "name of the input XML file")
	flag.StringVar(&localMedia, "localmedia", "", "url of the local media section")
	flag.Parse()

	fmt.Println("flags:", outdir, xmlFilename)

	if len(outdir) == 0 || len(xmlFilename) == 0 {
		log.Fatalf("flags missing")
	}

	// Open our xmlFile
	xmlFile, err := os.Open(xmlFilename)
	if err != nil {
		log.Fatalf("could not open file %s: %v", xmlFilename, err)
	}

	fmt.Println("Successfully opened:", xmlFilename)
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		log.Fatalf("could not read xml: %v", err)
	}

	var doc rss
	err = xml.Unmarshal(byteValue, &doc)
	if err != nil {
		log.Fatalf("could not parse xml: %v", err)
	}

	renderer := contentRenderer{
		transformContent: substituteMediaRoot,
	}

	fmt.Println("read items:", len(doc.Items))

	itemsByKind := make(map[string][]item)

	for _, it := range doc.Items {
		_, found := itemsByKind[it.PostType]
		if !found {
			itemsByKind[it.PostType] = make([]item, 0)
		}
		itemsByKind[it.PostType] = append(itemsByKind[it.PostType], it)
	}

	for k, items := range itemsByKind {
		if k == "attachment" {
			continue
		}
		fmt.Println(k, len(items))
		err := os.Mkdir(filepath.Join(outdir, k), 0750)
		if err != nil {
			log.Fatalf("could not create dir: %v", err)
		}
		fmt.Println("created dir", filepath.Join(outdir, k))
		for _, it := range items {
			if len(it.Slug) == 0 {
				continue
			}
			name := it.Slug
			if k == "post" {
				dt, err := time.Parse("2006-01-02 15:04:05", it.PostDate)
				if err == nil {
					name = fmt.Sprintf("%s/%s", dt.Format("2006/01/02"), it.Slug)
				}
			}
			err = os.MkdirAll(filepath.Join(outdir, k, name), 0750)
			if err != nil {
				log.Fatalf("could not create dir: %v", err)
			}
			fmt.Println("created dir", filepath.Join(outdir, k, name))

			f, err := os.Create(filepath.Join(outdir, k, name, "index.md"))
			if err != nil {
				log.Fatalf("could not create file: %v", err)
			}
			err = renderer.toMarkdown(it, f)
			if err != nil {
				log.Println("could not write post: ", err)
			}
			err = f.Sync()
			if err != nil {
				log.Println("could not flush file: ", err)
			}
			err = f.Close()
			if err != nil {
				log.Println("could not close file: ", err)
			}

			if len(it.Comments) > 0 {
				f, err := os.Create(filepath.Join(outdir, k, name, "comments.html"))
				if err != nil {
					log.Fatalf("could not create file: %v", err)
				}
				err = renderer.renderThreads(f, threadComments(it.Comments))
				if err != nil {
					log.Println("ayyyyssss", err)
				}
				err = f.Sync()
				if err != nil {
					log.Println("could not flush file: ", err)
				}
				err = f.Close()
				if err != nil {
					log.Println("could not close file: ", err)
				}
			}
		}
	}
}

// threadComments takes a list of comments, sorts through their
// family tree using the `ParentID` and `ID` attributes, and converts them
// to a list of threads where each node has its Children threads
//
// NOTE: assumes the comments form a tree
func threadComments(comments []comment) []commentThread {
	roots := make([]commentThread, 0, len(comments))
	for _, c := range comments {
		if c.ParentID == 0 {
			roots = append(roots, threadCommentLevel(c, comments))
		}
	}

	return roots
}

func threadCommentLevel(node comment, comments []comment) commentThread {
	var threads []commentThread
	for _, c := range comments {
		if c.ParentID == node.ID {
			threads = append(threads, threadCommentLevel(c, comments))
		}
	}
	var re commentThread
	re.comment = node
	re.Children = threads
	return re
}

func substituteMediaRoot(content string) string {
	replacer := strings.NewReplacer("http://plazamoyua.files.wordpress.com", "http://localhost:1313/media",
		"http://plazamoyua.com", "http://localhost:1313",
		"http://plazamoyua.wordpress.com", "http://localhost:1313",
		":cry:", "😥",
		":shock:", "😯",
		":grin:", "😀",
		":razz:", "😛",
		":P", "😛",
		":)", "🙂",
		";)", "😉",
		":wink:", "😉",
		":lol:", "😆",
		":arrow:", "➡",
		":twisted:", "😈",
		":idea:", "💡",
		":evil:", "👿",
		":oops:", "😳",
		":roll:", "🙄")
	return replacer.Replace(content)
}

// contentRenderer contains methods to transform and render the XML content
// into other formats
type contentRenderer struct {
	transformContent func(string) string
}

func escapeTitleQuotes(s string) string {
	r := strings.NewReplacer("\"", "\\\"")
	return r.Replace(s)
}

// toMarkdown adds a Hugo/jekyll front matter and displays a post/page as
// markdown
func (cr contentRenderer) toMarkdown(i item, writer io.Writer) error {
	var content string
	for _, enc := range i.Encodeds {
		if enc.XMLName.Space == "http://purl.org/rss/1.0/modules/content/" {
			content = cr.transformContent(enc.Data)
		}
	}

	var tt textTpl.Template

	var (
		tags       []string
		categories []string
	)
	for _, ct := range i.Categories {
		switch ct.Domain {
		case "category":
			categories = append(categories, ct.NiceName)
		case "post_tag":
			tags = append(tags, ct.NiceName)
		default:
			continue
		}
	}

	var categoriesLine, tagsLine string
	if len(categories) > 0 {
		categoriesLine = fmt.Sprintf(`categories: ["%s"]`, strings.Join(categories, "\", \""))
	}
	if len(tags) > 0 {
		tagsLine = fmt.Sprintf(`tags: ["%s"]`, strings.Join(tags, "\", \""))
	}

	d := struct {
		Title          string
		PubDate        string
		Author         string
		Content        string
		Slug           string
		Link           string
		CategoriesLine string
		TagsLine       string
	}{
		Title:          escapeTitleQuotes(i.Title),
		PubDate:        i.PubDate,
		Author:         i.Author,
		Content:        content,
		Slug:           i.Slug,
		Link:           i.Link,
		CategoriesLine: categoriesLine,
		TagsLine:       tagsLine,
	}

	t, err := tt.Parse(`
---
title: "{{.Title }}"
date: "{{.PubDate}}"
author: "{{.Author}}"
original: {{.Link}}
slug: "{{.Slug}}"
{{.CategoriesLine}}
{{.TagsLine}}
---

{{.Content}}`)

	if err != nil {
		return err
	}
	return t.Execute(writer, d)
}

// threadToHTML renders a single thread as HTML, starting a new
// sub ordered-list fore each parent/child generation
// NOTE: Markdown could accomodate this too, but being whitespace-sensitive,
// this makes it an inconvenient choice. HTML is the better format for code-gen
func (cr contentRenderer) threadToHTML(thread commentThread) (template.HTML, error) {
	t, err := template.New("tpl").Parse(`
	<li>
	<div class="comment">
		<span class="author">{{.AuthorName}}</span>
		<span class="date">{{.CommentDate}}</span>
		<div>
			{{.Content}}
		</div>
		{{ with .ChildrenHTML}}
		<div class="children">
			<ul>
			{{ range . }}
				{{ . }}
			{{ end }}
			</ul>
		</div>
		{{ end }}
	</div>
	</li>
`)
	if err != nil {
		log.Fatalf("bad template: %v", err)
	}

	if len(thread.Children) == 0 {
		buffer := bytes.Buffer{}
		thread.Content = template.HTML(cr.transformContent((string(thread.Content))))
		err = t.Execute(&buffer, thread)
		if err != nil {
			return "", err
		}
		return template.HTML(buffer.String()), nil
	}

	for _, child := range thread.Children {
		ht, err := cr.threadToHTML(child)
		if err != nil {
			return "", err
		}
		thread.ChildrenHTML = append(thread.ChildrenHTML, ht)
	}
	buffer := bytes.Buffer{}
	err = t.Execute(&buffer, thread)
	if err != nil {
		return "", err
	}

	return template.HTML(buffer.String()), err
}

// renderThreads goes overa all the comment threads and renders them to the
// appropriate writer/file/buffer
func (cr contentRenderer) renderThreads(writer io.Writer, comments []commentThread) error {
	_, err := writer.Write([]byte("<div class=\"comments\"><ul>\n"))
	if err != nil {
		return err
	}
	for _, c := range comments {
		ht, err := cr.threadToHTML(c)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte(ht))
		if err != nil {
			return err
		}
	}
	_, err = writer.Write([]byte("</ul></div>\n"))
	if err != nil {
		return err
	}
	return nil
}
