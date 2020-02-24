package main

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

const (
	codeReviewPrefix = "CR"
)

func trimTabs(s string, n int) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		n--
		return unicode.IsSpace(r) && n >= 0
	})
}

func sanitizeCommentary(text string, column int) string {
	text = strings.TrimPrefix(text, codeReviewPrefix)
	text = strings.TrimPrefix(text, ":")
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = trimTabs(line, column-1)
	}
	return strings.Join(lines, "\n")
}

type fileComments struct {
	fileName string
	fileBody string
	comments []comment
}

type comment struct {
	text       string
	line       int
	start, end int
}

func parseFile(path string) ([]comment, error) {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, errors.Wrap(err, "parse file AST")
	}

	offset := 0
	var res []comment
	for _, comm := range f.Comments {
		text := strings.TrimSpace(comm.Text())

		if !strings.HasPrefix(text, codeReviewPrefix) {
			continue
		}

		pos := fset.Position(comm.Pos())
		end := fset.Position(comm.End())
		res = append(res, comment{
			text:  sanitizeCommentary(text, pos.Column),
			line:  pos.Line - offset,
			start: pos.Line,
			end:   end.Line,
		})
		offset += end.Line - pos.Line + 1
	}

	return res, nil
}

func main() {

	// https://gitlab.ozon.ru/smalenkov/playground/merge_requests/1

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		log.Fatal("empty auth token: please set GITLAB_TOKEN env variable")
	}
	baseURL := os.Getenv("GITLAB_BASE_URL")

	projectPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		log.Fatal("usage: gitlab-review <link to the merge request>")
	}
	mrLink := os.Args[1]
	mrLink = strings.TrimPrefix(mrLink, baseURL)
	toks := strings.Split(mrLink, "/merge_requests/")
	if len(toks) < 2 {
		log.Fatal("malformed merge request link")
	}
	projectID := toks[0]
	mrID, err := strconv.Atoi(toks[1])
	if err != nil {
		log.Fatal(errors.Wrap(err, "parse merge request ID"))
	}

	client := newClient(token, baseURL)
	err = client.init(projectID, mrID)
	if err != nil {
		log.Fatal(err)
	}

	var filesComments []fileComments

	filepath.Walk(projectPath, filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(info.Name()) != ".go" {
			return nil
		}
		fname, err := filepath.Rel(projectPath, path)
		if err != nil {
			return errors.Wrap(err, "get relative path")
		}

		comments, err := parseFile(path)
		if err != nil {
			return errors.Wrap(err, "parse file")
		}
		if len(comments) == 0 {
			return nil
		}

		raw, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "read file body")
		}
		filesComments = append(filesComments, fileComments{
			fileName: fname,
			fileBody: string(raw),
			comments: comments,
		})
		return nil
	}))

	err = newView(client, filesComments)
	if err != nil {
		log.Fatal(err)
	}
}
