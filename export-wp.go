package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

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
		transformContent: cleanContent,
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

	for kind, items := range itemsByKind {
		if kind == "attachment" || kind == "nav_menu_item" {
			continue
		}
		fmt.Println(kind, len(items))
		err := os.Mkdir(filepath.Join(outdir, kind), 0750)
		if err != nil {
			log.Fatalf("could not create dir: %v", err)
		}
		fmt.Println("created dir", filepath.Join(outdir, kind))
		for _, it := range items {
			if len(it.Slug) == 0 {
				continue
			}
			if it.Status == "trash" || it.Status == "trashed" || it.Status == "draft" {
				continue
			}
			name := it.Slug
			if kind == "post" {
				dt, err := time.Parse("2006-01-02 15:04:05", it.PostDate)
				if err == nil {
					name = filepath.Join(dt.Format("2006/01/02"), it.Slug)
				}
			}
			err = os.MkdirAll(filepath.Join(outdir, kind, name), 0750)
			if err != nil {
				log.Fatalf("could not create dir: %v", err)
			}
			fmt.Println("created dir", filepath.Join(outdir, kind, name))

			f, err := os.Create(filepath.Join(outdir, kind, name, "index.md"))
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
				f, err := os.Create(filepath.Join(outdir, kind, name, "comments.html"))
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
