package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"regexp"
	"strings"
	textTpl "text/template"
)

// threadComments takes a list of comments, sorts through their
// family tree using the `ParentID` and `ID` attributes, and converts them
// to a list of threads where each node has its Children threads
//
// NOTE: assumes the comments form a tree
func threadComments(comments []comment) []commentThread {
	commentsWithParentID := make(map[int][]comment)
	for _, cm := range comments {
		if cm.Approved != "trash" {
			commentsWithParentID[cm.ParentID] = append(commentsWithParentID[cm.ParentID], cm)
		}
	}

	roots := make([]commentThread, 0, len(commentsWithParentID[0]))
	for _, c := range commentsWithParentID[0] {
		roots = append(roots, threadCommentLevel(c, commentsWithParentID))
	}

	return roots
}

func threadCommentLevel(node comment, commentsWithParentID map[int][]comment) commentThread {
	threads := make([]commentThread, len(commentsWithParentID[node.ID]))
	for i, c := range commentsWithParentID[node.ID] {
		threads[i] = threadCommentLevel(c, commentsWithParentID)
	}
	var re commentThread
	re.comment = node
	re.Children = threads
	return re
}

// linkifyText finds "free" urls in text (i.e. urls not in an <a href=""> context), and
// puts them inside an <a>
//
// Main link matcher regex inspired by
// https://stackoverflow.com/questions/26561149/golang-regex-to-find-urls-in-a-string
func linkifyText(in string) (string, error) {
	re, err := regexp.Compile(`(\s+)((http|ftp|https)://([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?)`)
	if err != nil {
		return "", err
	}

	return re.ReplaceAllString(in, `$1<a href="$2">$2</a>`), nil
}

// cleanLink makes self-links potable for use in site page URL's
func cleanLink(link string) string {
	replacer := strings.NewReplacer("http://plazamoyua.com", "",
		"http://plazamoyua.wordpress.com", "",
		"https://plazamoyua.com", "",
		"https://plazamoyua.wordpress.com", "",
	)
	return replacer.Replace(link)
}

// cleanContent scrubs content text
//  - make self-references portable
//  - substitute emoticon shortcodes for actual emoticon Unicodes
func cleanContent(content string) string {
	replacer := strings.NewReplacer("http://plazamoyua.files.wordpress.com", "/media",
		"https://plazamoyua.files.wordpress.com", "/media",
		"http://plazamoyua.com/tag/", "/tags/",
		"http://plazamoyua.wordpress.com/tag/", "/tags/",
		"http://plazamoyua.com/category/", "/categories/",
		"http://plazamoyua.wordpress.com/category/", "/categories/",
		"http://plazamoyua.com/", "/",
		"http://plazamoyua.wordpress.com/", "/",
		"https://plazamoyua.com/tag/", "/tags/",
		"https://plazamoyua.wordpress.com/tag/", "/tags/",
		"https://plazamoyua.com/category/", "/categories/",
		"https://plazamoyua.wordpress.com/category/", "/categories/",
		"https://plazamoyua.com/", "/",
		"https://plazamoyua.wordpress.com/", "/",
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

	data := struct {
		Title          string
		PubDate        string
		Author         string
		Content        string
		Slug           string
		Link           string
		CategoriesLine string
		TagsLine       string
		URL            string
	}{
		Title:          escapeTitleQuotes(i.Title),
		PubDate:        i.PubDate,
		Author:         i.Author,
		Content:        content,
		Slug:           i.Slug,
		Link:           i.Link,
		URL:            cleanLink(i.Link),
		CategoriesLine: categoriesLine,
		TagsLine:       tagsLine,
	}

	if !strings.Contains(data.URL, data.Slug) {
		data.URL = data.Slug
		fmt.Printf("WARN: disregarding item URL %s, using slug: %s\n", data.URL, data.Slug)
	}

	t, err := tt.Parse(`
---
title: "{{.Title }}"
date: "{{.PubDate}}"
author: "{{.Author}}"
original: {{.Link}}
slug: "{{.Slug}}"
url: "{{.URL}}"
{{.CategoriesLine}}
{{.TagsLine}}
---

{{.Content}}`)

	if err != nil {
		return err
	}
	return t.Execute(writer, data)
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

	// if possible, make free urls in comments into links
	linkified, err := linkifyText(string(thread.Content))
	if err != nil {
		linkified = string(thread.Content)
	}
	thread.Content = template.HTML(cr.transformContent(linkified))
	buffer := bytes.Buffer{}
	if len(thread.Children) == 0 {
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
