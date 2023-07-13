package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	template2 "html/template"

	"github.com/PuerkitoBio/goquery"
	"github.com/avelino/awesome-go/pkg/markdown"
)

const awesomeGoReadmePath = "https://raw.githubusercontent.com/avelino/awesome-go/main/README.md"
const outputDir = "out/"

const githubApiUrl = "https://api.github.com/repos/"

var tplIndex = template.Must(template.ParseFiles("tmpl/parse_index.tmpl.html"))
var outIndexFilePath = filepath.Join(outputDir, "parse_index.html")

type Category struct {
	Name        string      `json:"category_name"`
	Description string      `json:"category_description"`
	Links       *[]Link     `json:"links,omitempty"`
	Children    *[]Category `json:"sub_categories,omitempty"`
}

type Link struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Stars       int64  `json:"stargazers_count"`
	Forks       int64  `json:"forks_count"`
	OpenIssues  int64  `json:"open_issues_count"`
	Watchers    int64  `json:"watchers_count"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	PushedAt    int64  `json:"pushed_at"`
	Achived     bool   `json:"achived"`
	Disabled    bool   `json:"disabled"`
}

func main() {
	if err := buildStaticSite(); err != nil {
		panic(err)
	}
}

func buildStaticSite() error {
	if err := removeOutputDir(outputDir); err != nil {
		return err
	}

	if err := mkdirAll(outputDir); err != nil {
		return err
	}

	if err := renderIndex(outIndexFilePath); err != nil {
		return err
	}

	parseIndex, err := os.ReadFile(outIndexFilePath)
	if err != nil {
		return fmt.Errorf("read converted html: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(parseIndex))
	if err != nil {
		return fmt.Errorf("create goquery instance: %w", err)
	}

	extractCategories(doc)

	return nil
}

func extractCategories(doc *goquery.Document) error {
	var rootErr error

	doc.
		Find("body #contents").
		NextFiltered("ul").
		Find("ul").
		EachWithBreak(func(_ int, selUl *goquery.Selection) bool {
			if rootErr != nil {
				return false
			}

			selUl.Find("li a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
				selector, exsist := a.Attr("href")
				if !exsist {
					return true
				}

				doc.Find(selector).EachWithBreak(func(_ int, header *goquery.Selection) bool {
					fmt.Println(header.Html())
					return true
				})
				return true
			})
			/*

				selUl.
					Find("li a").
					EachWithBreak(func(_ int, s *goquery.Selection) bool {
						selector, exists := s.Attr("href")
						if !exists {
							return true
						}

						category, err := extractCategory(doc, selector)
						if err != nil {
							rootErr = fmt.Errorf("extract category: %w", err)
							return false
						}

						categories[selector] = *category

						return true
					})
			*/
			return true
		})

	if rootErr != nil {
		return fmt.Errorf("extract categories: %w", rootErr)
	}

	return nil
}

//func extractCategory()

/*
	func extractCategory(doc *goquery.Document, selector string) (*Category, error) {
		var category Category
		var err error

		doc.Find(selector).EachWithBreak(func(_ int, selCatHeader *goquery.Selection) bool {
			selDescr := selCatHeader.NextFiltered("p")
			// FIXME: bug. this would select links from all neighboring
			//   sub-categories until the next category. To prevent this we should
			//   find only first ul
			ul := selCatHeader.NextFilteredUntil("ul", "h2")

			var links []Link
			ul.Find("li").Each(func(_ int, selLi *goquery.Selection) {
				selLink := selLi.Find("a")
				url, _ := selLink.Attr("href")
				link := Link{
					Title: selLink.Text(),
					// FIXME(kazhuravlev): Title contains only title but
					// 	description contains Title + description
					Description: selLi.Text(),
					URL:         url,
				}
				links = append(links, link)
			})

			// FIXME: In this case we would have an empty category in main index.html with link to 404 page.
			if len(links) == 0 {
				err = errors.New("category does not contain links")
				return false
			}

			category = Category{
				Slug:        slug.Generate(selCatHeader.Text()),
				Title:       selCatHeader.Text(),
				Description: selDescr.Text(),
				Links:       links,
			}

			return true
		})

		if err != nil {
			return nil, fmt.Errorf("build a category: %w", err)
		}

		return &category, nil
	}
*/
func removeOutputDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("[err] remove dir: %w", err)
	}

	return nil
}

func mkdirAll(path string) error {
	_, err := os.Stat(path)
	// directory is exists
	if err == nil {
		return nil
	}

	// unexpected error
	if !os.IsNotExist(err) {
		return fmt.Errorf("unexpected result of dir stat: %w", err)
	}

	// directory is not exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("midirAll: %w", err)
	}

	return nil
}

// renderIndex generate site html (index.html) from markdown file
func renderIndex(outFilename string) error {
	resp, err := http.Get(awesomeGoReadmePath)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	input, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	body, err := markdown.ToHTML(input)
	if err != nil {
		return err
	}

	f, err := os.Create(outFilename)
	if err != nil {
		return err
	}

	fmt.Printf("Write Index file: %s\n", outIndexFilePath)
	data := map[string]interface{}{
		"Body": template2.HTML(body),
	}
	if err := tplIndex.Execute(f, data); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close index file: %w", err)
	}

	return nil
}
