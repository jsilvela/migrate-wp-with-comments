package main

import (
	"encoding/xml"
	"html/template"
)

// rss is the top-level XML element in the WordPress export
type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []item   `xml:"channel>item"`
}

// item is the place where posts, pages and attachments are represented
type item struct {
	XMLName       xml.Name
	Categories    []category `xml:"category"`
	Link          string     `xml:"link"`
	PubDate       string     `xml:"pubDate"`
	Title         string     `xml:"title"`
	Encodeds      []encoded  `xml:"encoded"`        // space: content / excerpt
	Author        string     `xml:"creator"`        // space: dc
	PostDate      string     `xml:"post_date"`      // space: wp
	Slug          string     `xml:"post_name"`      // space: wp
	PostDateGMT   string     `xml:"post_date_gmt"`  // space: wp
	PostMeta      postMeta   `xml:"postmeta"`       // space: wp
	Comments      []comment  `xml:"comment"`        // space: wp
	ID            int        `xml:"post_id"`        // space: wp
	CommentStatus string     `xml:"comment_status"` // space: wp - may be open, closed
	PostParent    int        `xml:"post_parent"`    // space: wp
	PostType      string     `xml:"post_type"`      // space: wp
	Status        string     `xml:"status"`         // space: wp - may be publish, inherit, trash, draft ...
}

// category represents a category or tag
type category struct {
	XMLName  xml.Name
	Domain   string `xml:"domain,attr"` // values: 'category' / 'post_tag'
	NiceName string `xml:"nicename,attr"`
	Data     string `xml:",cdata"`
}

// encoded represents the payload of an Item
// may be in Space 'content' or 'excerpt'
type encoded struct {
	XMLName xml.Name
	Data    string `xml:",cdata"`
}

// postMeta represents WordPress metadata
// Space: wp
type postMeta struct {
	XMLName   xml.Name
	MetaKey   string    `xml:"meta_key"`
	MetaValue metaValue `xml:"meta_value"`
}

// metaValue content of a postMeta
// Space: wp
type metaValue struct {
	XMLName xml.Name
	Value   string `xml:",cdata"`
}

// comment represents a comment on the site, not an XML comment
// the payload `Content` may contain embedded HTML, and is assumed to be
// safe for inclusion into an HTML document
// Space: wp
type comment struct {
	XMLName        xml.Name
	Approved       string        `xml:"comment_approved"` // may be: 1, trash
	AuthorName     string        `xml:"comment_author"`
	AuthorEmail    string        `xml:"comment_author_email"`
	AuthorURL      string        `xml:"comment_author_url"`
	Content        template.HTML `xml:"comment_content"`
	ID             int           `xml:"comment_id"`
	ParentID       int           `xml:"comment_parent"`
	CommentDate    string        `xml:"comment_date"`
	CommentDateGMT string        `xml:"comment_date_gmt"`
}

// commentThread represents a comment and its descendants
type commentThread struct {
	comment
	Children     []commentThread
	ChildrenHTML []template.HTML // this is just a convenience for rendering
}
