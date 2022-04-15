# Export WordPress to Hugo preserving comment threads

Yet another WordPress-to-Hugo export tool, but one which serves comment threads
statically using Hugo, instead of relying on disqus or other
external comment software.

Written from scratch in Go.

## Using

Download, clone etc. and `cd` into the directory created.

``` sh
% go test
% go build
% ./migrate-wp -h
% mkdir exported
% ./migrate-wp -outdir exported -xmlfile myWPexport.xml
```

## How?

WordPress XML exports include a flat list of comments for each page. Each comment
has an `ID`, and a `ParentID`. A `ParentID` of 0 indicates the comment hangs
directly from the post/page. A `ParentID` pointing to another comment's `ID` is
a response to that comment.
Comment threads of arbitrary depth can be built with the `ParentID` linkage.

This tool will build a tree data structure with the comment threads, and write
them into an HTML file, using unordered lists to build display hierarchies.
This HTML file can then be embedded into the page content.

A page with comments can be represented as:

``` sh
post/my-post/
  index.md
  comments.html
```

Then, an appropriate **layout** can combine the contents:

``` html
<main>
    …
	{{ .Content }}
    <hr/>
    {{ with .Resources.GetMatch "comments.html" }}
        {{ .Content | safeHTML }}
    {{ end }}
</main>
```

Please refer to [hugo documentation.](https://gohugo.io/templates/lookup-order/)

## Why?

In March 2022 I had a project to move a blog away from Wordpress, into a format
that could be preserved in a thumb drive, read from the file-system, or served
easily as a static site, locally or on an public site.

My tool of choice to generate static sites is [hugo](https://gohugo.io/).

The better way to migrate a site from WordPress is to get into the site's
WP administration panel, and use the Export options.
This can produce:

- A full dump of the text of the site, into one or more XML files. This includes
  posts, pages, and the reader comments inside them.
- A full dump of the media library that was hosted in the site, which could
  include any images, `.docx`, PDF or other media that were saved and served
  from the site.

The XML files are unwieldy, and it is convenient to convert the content in them
into Markdown files with a metadata section, to be used with some static
site generator.

There are quite a few tools to help with this, but they are all limited,
just good enough for the particular needs of their creators.
None of them handled my particular needs:

1. The pages should be generated **with** the comments
2. The comments should be threaded
3. Any media files stored with the original site should be served locally from
  the statically generated version

Those existing tools that didn't disregard comments, gathered them to be served
by https://disqus.com or other solutions, and left to them comment threading
and display.

I wanted the comments frozen, closed, threaded, and served statically.

I have taken inspiration from two main tools:

- [wordpress-export-to-markdown](https://github.com/lonekorean/wordpress-export-to-markdown) (javascript)
- [`wordpressxml2jekyll.rb`](https://gist.github.com/markoa/268428) (ruby)

The first one is very usable and complete, but does not handle comments.
The second does handle comments, but puts them in a `disqus`-format YAML file,
and delegates.

Since I was going to have to do quite a bit of coding, I decided to build from
scratch using Go.
