package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	textTpl "text/template"
	"time"
)

const (
	WORDPRESS_XML_FILE_PATH = "plazamoyuacom.wordpress.2022-03-07.000.xml" // "foo.xml"
	OUTPUT_PATH             = "export3"
	ORIGINAL_DOMAIN         = "https://plazamoyua.com"
	FILES                   = "http://plazamoyua.files.wordpress.com/"
)

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []item   `xml:"channel>item"`
}

// item is the place where posts, pages and attachments are represented
type item struct {
	XMLName       xml.Name
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	PubDate       string    `xml:"pubDate"`
	Author        string    `xml:"creator"`       // space: dc
	PostDate      string    `xml:"post_date"`     // space: wp
	Slug          string    `xml:"post_name"`     // space: wp
	PostDateGMT   string    `xml:"post_date_gmt"` // space: wp
	Encodeds      []encoded `xml:"encoded"`       // space: content
	PostMeta      postMeta  `xml:"postmeta"`
	Comments      []comment `xml:"comment"`
	ID            int       `xml:"post_id"`
	CommentStatus string    `xml:"comment_status"`
	PostParent    int       `xml:"post_parent"`
	PostType      string    `xml:"post_type"`
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
	Id             int           `xml:"comment_id"`
	ParentId       int           `xml:"comment_parent"`
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

	// Open our xmlFile
	xmlFile, err := os.Open(WORDPRESS_XML_FILE_PATH)
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}

	fmt.Println("Successfully opened:", WORDPRESS_XML_FILE_PATH)
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
		err := os.Mkdir(filepath.Join(OUTPUT_PATH, k), 0750)
		if err != nil {
			log.Fatalf("could not create dir: %v", err)
		}
		fmt.Println("created dir", filepath.Join(OUTPUT_PATH, k))
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
			err = os.MkdirAll(filepath.Join(OUTPUT_PATH, k, name), 0750)
			if err != nil {
				log.Fatalf("could not create dir: %v", err)
			}
			fmt.Println("created dir", filepath.Join(OUTPUT_PATH, k, name))

			f, err := os.Create(filepath.Join(OUTPUT_PATH, k, name, "index.md"))
			if err != nil {
				log.Fatalf("could not create file: %v", err)
			}
			err = it.toMarkdown(f)
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
				f, err := os.Create(filepath.Join(OUTPUT_PATH, k, name, "comments.html"))
				if err != nil {
					log.Fatalf("could not create file: %v", err)
				}
				err = renderThreads(f, threadComments(it.Comments))
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

// toMarkdown adds a Hugo/jekyll front matter and displays a post/page as
// markdown
func (i item) toMarkdown(writer io.Writer) error {
	var content string
	for _, enc := range i.Encodeds {
		if enc.XMLName.Space == "http://purl.org/rss/1.0/modules/content/" {
			content = enc.Data
		}
	}

	var tt textTpl.Template

	d := struct {
		Title   string
		PubDate string
		Author  string
		Content string
		Slug    string
	}{
		Title:   i.Title,
		PubDate: i.PubDate,
		Author:  i.Author,
		Content: content,
		Slug:    i.Slug,
	}

	t, err := tt.Parse(`
---
title: "{{.Title | html }}"
date: "{{.PubDate}}"
author: "{{.Author}}"
slug: "{{.Slug}}"
---

{{.Content}}`)

	if err != nil {
		return err
	}
	return t.Execute(writer, d)
}

// threadComments takes a list of comments, sorts through their
// family tree using the `ParentID` and `ID` attributes, and converts them
// to a list of threads where each node has its Children threads
//
// NOTE: assumes the comments form a tree
func threadComments(comments []comment) []commentThread {
	roots := make([]commentThread, 0, len(comments))
	for _, c := range comments {
		if c.ParentId == 0 {
			roots = append(roots, threadCommentLevel(c, comments))
		}
	}

	return roots
}

func threadCommentLevel(node comment, comments []comment) commentThread {
	var threads []commentThread
	for _, c := range comments {
		if c.ParentId == node.Id {
			threads = append(threads, threadCommentLevel(c, comments))
		}
	}
	var re commentThread
	re.comment = node
	re.Children = threads
	return re
}

func commentsToHTML(writer io.Writer, comments []comment) error {

	var tt template.Template
	t, err := tt.Parse(`
	<li>
	<div>
		<span>{{.AuthorName}}</span>
		<span>{{.Id}} < {{.ParentId}}</span>
		<div>
		{{.Content}}
		</div>
	</div>
	</li>`)
	if err != nil {
		log.Fatalf("bad template: %v", err)
	}

	for _, cm := range comments {
		err = t.Execute(writer, cm)
		if err != nil {
			return err
		}
	}
	return nil
}

// threadToHTML renders a single thread as HTML, starting a new
// sub ordered-list fore each generation
// NOTE: Markdown could accomodate this too, but being whitespace-sensitive,
// this makes it an inconvenient choice. HTML proves the better format for code-gen
func threadToHTML(thread commentThread) (template.HTML, error) {
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
		err = t.Execute(&buffer, thread)
		if err != nil {
			return "", err
		}
		return template.HTML(buffer.String()), nil
	}

	for _, child := range thread.Children {
		ht, err := threadToHTML(child)
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
func renderThreads(writer io.Writer, comments []commentThread) error {
	_, err := writer.Write([]byte("<div class=\"comments\"><ul>\n"))
	if err != nil {
		return err
	}
	for _, c := range comments {
		ht, err := threadToHTML(c)
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
