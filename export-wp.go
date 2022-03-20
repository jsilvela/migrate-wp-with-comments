package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	WORDPRESS_XML_FILE_PATH = "foo.xml" // "plazamoyuacom.wordpress.2022-03-07.000.xml"
	OUTPUT_PATH             = "export2"
	ORIGINAL_DOMAIN         = "https://plazamoyua.com"
)

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []Item   `xml:"channel>item"`
}

type Author struct {
	XMLName xml.Name `xml:"wp:author"`
	// 	<wp:author_id>14109008</wp:author_id>
	// 	<wp:author_login>lbouza</wp:author_login>
	// 	<wp:author_email>lbouza@telefonica.net</wp:author_email>
	// 	<wp:author_display_name><![CDATA[Luis Bouza-Brey]]></wp:author_display_name>
	// 	<wp:author_first_name><![CDATA[Luis]]></wp:author_first_name>
	// 	<wp:author_last_name><![CDATA[Bouza-Brey]]></wp:author_last_name>
	//
}

// type Title struct {
// 	XMLName xml.Name `xml:"title"`
// }

// type Link struct {
// 	XMLName xml.Name `xml:"link"`
// }

// type description struct {
// 	XMLName xml.Name `xml:"description"`
// }

// type pubDate struct {
// 	XMLName xml.Name `xml:"pubDate"`
// }

// type language struct {
// 	XMLName xml.Name `xml:"language"`
// }

// type wxrVersion struct {
// 	XMLName xml.Name `xml:"wp:wxr_version"`
// }

// type wpBaseSiteUrl struct {
// 	XMLName xml.Name `xml:"wp:base_site_url"`
// }

// type wpBaseBlogUrl struct {
// 	XMLName xml.Name `xml:"wp:base_blog_url"`
// }

// class Post
//   attr_accessor :title, :post_date, :created_at, :slug, :url, :content, :textile_content
//   attr_accessor :hpricot_element

// class Comment
//   attr_accessor :author_name, :author_email, :author_url, :content, :post, :id, :parent_id

// our struct which contains the complete
// array of all Users in the file
type Item struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	PubDate     string   `xml:"pubDate"`
	Author      string   `xml:"creator"`   // space: dc
	PostDate    string   `xml:"post_date"` // space: wp
	CreatedAt   time.Time
	Slug        string `xml:"post_name"`     // space: wp
	PostDateGMT string `xml:"post_date_gmt"` // space: wp
	// Encodeds    []content `xml:">encoded"`
	Encodeds []encoded `xml:"encoded"` // space: content
	// Excerpt     content `xml:"excerpt:encoded"` // space: excerpt
	PostMeta postMeta
	// Description content // space: description
}

type encoded struct {
	XMLName xml.Name
	Data    string `xml:",cdata"`
}

type postMeta struct {
	XMLName   xml.Name `xml:"postmeta"` // namespace: wp
	MetaKey   string   `xml:"meta_key"`
	MetaValue metaValue
}

type metaValue struct {
	XMLName xml.Name `xml:"meta_value"`
	Value   string   `xml:",cdata"`
}

// type creator struct {
// 	XMLName xml.Name `xml:"creator"`
// 	Content string
// }

type Comment struct {
	PostUrl     string
	AuthorName  string `xml:"wp:comment_author"`
	AuthorEmail string `xml:"wp:comment_author_email"`
	AuthorURL   string `xml:"wp:comment_author_url"`
	Content     string `xml:"wp:comment_content"`
	Id          string `xml:"wp:comment_id"`
	ParentId    string `xml:"wp:comment_parent"`

	CommentDate string `xml:"wp:comment_date_gmt"`
	CreatedAt   time.Time
	XMLName     xml.Name `xml:"wp:comment"`
}

func main() {

	// Open our xmlFile
	xmlFile, err := os.Open(WORDPRESS_XML_FILE_PATH)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}

	fmt.Println("Successfully Opened file")
	// defer the closing of our xmlFile so that we can parse it later on
	defer xmlFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		log.Fatalf("could not read xml: %v", err)
	}

	// we initialize our Users array
	var doc rss
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(byteValue, &doc)
	if err != nil {
		log.Fatalf("could not read xml: %v", err)
	}

	fmt.Println("read items:", len(doc.Items))
	// fmt.Printf("read stuff: %#v\n", doc)

	// we iterate through every user within our users array and
	// print out the user Type, their name, and their facebook url
	// as just an example
	for i := 0; i < len(doc.Items); i++ {
		fmt.Println("title: " + doc.Items[i].Title)
		fmt.Printf("%#v\n", doc.Items[i])
	}

	// out, err := xml.Marshal(doc)
	// if err != nil {
	// 	log.Fatalf("could not marshal out: %v", err)
	// }
	// fmt.Println(string(out))
}
