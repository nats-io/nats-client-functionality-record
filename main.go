package main

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"gitlab.com/golang-commonmark/markdown"
)

type ADRMeta struct {
	Index   int
	Authors []string
	Date    time.Time
	Status  string
	Tags    []string
	Path    string
}

type ADR struct {
	Heading string
	Meta    ADRMeta
}

var (
	validStatus = []string{"Approved", "Partially Implemented", "Implemented", "Rejected"}
)

func parseCommaList(l string) []string {
	tags := strings.Split(l, ",")
	res := []string{}
	for _, t := range tags {
		res = append(res, strings.TrimSpace(t))
	}
	return res
}

func parseADR(path string) (*ADR, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	md := markdown.New()
	tokens := md.Parse(body)

	in1stHdr := false
	in1stTbl := false
	metaSet := false
	curHdrKey := ""

	adr := ADR{
		Meta: ADRMeta{
			Path: path,
		},
	}

	for _, t := range tokens {
		switch tok := t.(type) {
		case *markdown.Inline:
			switch {
			case in1stHdr:
				adr.Heading = tok.Content

			case curHdrKey == "Index":
				adr.Meta.Index, err = strconv.Atoi(tok.Content)
				if err != nil {
					return nil, fmt.Errorf("invalid index number %q: %s", tok.Content, err)
				}

			case curHdrKey == "Date":
				t, err := time.Parse("2006-01-02", tok.Content)
				if err != nil {
					return nil, fmt.Errorf("invalid date format, not YYYY-MM-DD: %s", err)
				}

				adr.Meta.Date = t
			case curHdrKey == "Author":
				adr.Meta.Authors = parseCommaList(tok.Content)

			case curHdrKey == "Status":
				adr.Meta.Status = tok.Content

			case curHdrKey == "Tags":
				adr.Meta.Tags = parseCommaList(tok.Content)

			case in1stTbl:
				curHdrKey = tok.Content
			}

		case *markdown.TbodyOpen:
			if !metaSet {
				in1stTbl = true
			}

		case *markdown.HeadingOpen:
			if adr.Heading == "" {
				in1stHdr = true
			}

		case *markdown.TbodyClose:
			in1stTbl = false
		case *markdown.HeadingClose:
			in1stHdr = false
		case *markdown.TableClose:
			in1stTbl = false
		case *markdown.TrClose:
			curHdrKey = ""
		}
	}

	if adr.Meta.Index == 0 {
		return nil, fmt.Errorf("invalid ADR Index in %s", adr.Meta.Path)
	}
	if adr.Meta.Date.IsZero() {
		return nil, fmt.Errorf("date is required in %s", adr.Meta.Path)
	}
	if !isValidStatus(adr.Meta.Status) {
		return nil, fmt.Errorf("invalid status %q, must be one of: %s in %s", adr.Meta.Status, strings.Join(validStatus, ", "), adr.Meta.Path)
	}
	if len(adr.Meta.Authors) == 0 {
		return nil, fmt.Errorf("authors is required in %s", adr.Meta.Path)
	}
	if len(adr.Meta.Tags) == 0 {
		return nil, fmt.Errorf("tags is required in %s", adr.Meta.Path)
	}

	return &adr, nil
}

func isValidStatus(status string) bool {
	for _, s := range validStatus {
		if status == s {
			return true
		}
	}

	return false
}

func renderIndexes(adrs []*ADR) error {
	tags := map[string]int{}
	for _, adr := range adrs {
		for _, tag := range adr.Meta.Tags {
			tags[tag] = 1
		}
	}

	tagsList := []string{}
	for k := range tags {
		tagsList = append(tagsList, k)
	}
	sort.Strings(tagsList)

	type tagAdrs struct {
		Tag  string
		Adrs []*ADR
	}

	renderList := []tagAdrs{}

	for _, tag := range tagsList {
		matched := []*ADR{}
		for _, adr := range adrs {
			for _, mt := range adr.Meta.Tags {
				if tag == mt {
					matched = append(matched, adr)
				}
			}
		}

		renderList = append(renderList, tagAdrs{Tag: tag, Adrs: matched})
	}

	funcMap := template.FuncMap{
		"join": func(i []string) string {
			return strings.Join(i, ", ")
		}}

	readme, err := template.New(".readme.templ").Funcs(funcMap).ParseFiles(".readme.templ")
	if err != nil {
		return err
	}
	err = readme.Execute(os.Stdout, renderList)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	dir, err := os.ReadDir("adr")
	if err != nil {
		panic(err)
	}

	adrs := []*ADR{}

	for _, mdf := range dir {
		if mdf.IsDir() {
			continue
		}

		if path.Ext(mdf.Name()) != ".md" {
			continue
		}

		adr, err := parseADR(path.Join("adr", mdf.Name()))
		if err != nil {
			panic(err)
		}

		adrs = append(adrs, adr)
	}

	err = renderIndexes(adrs)
	if err != nil {
		panic(err)
	}
}
