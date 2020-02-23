package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/xanzy/go-gitlab"
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
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = trimTabs(line, column-1)
	}
	return strings.Join(lines, "\n")
}

type comment struct {
	text string
	file string
	line int
}

func parseFile(path, fname string) []comment {
	fset := token.NewFileSet() // positions are relative to fset

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return nil
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
			file: fname,
			text: sanitizeCommentary(text, pos.Column),
			line: pos.Line - offset,
		})
		log.Printf("start: %d, end: %d", pos.Line, end.Line)
		offset += end.Line - pos.Line + 1
	}

	return res
}

func main() {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		log.Fatal("empty auth token: please set GITLAB_TOKEN env variable")
	}
	baseURL := os.Getenv("GITLAB_BASE_URL")

	projectPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("path:", projectPath)

	var comments []comment

	filepath.Walk(projectPath, filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		log.Print(path)

		if filepath.Ext(info.Name()) != ".go" {
			return nil
		}
		fname, err := filepath.Rel(projectPath, path)
		if err != nil {
			log.Fatal(err)
		}
		comments = append(comments, parseFile(path, fname)...)
		return nil
	}))

	git := gitlab.NewClient(nil, token)
	git.SetBaseURL(baseURL)

	mr, _, err := git.MergeRequests.GetMergeRequest(3637, 1, nil)
	if err != nil {
		log.Fatal(err)
	}
	diffReffs := mr.DiffRefs

	for _, comm := range comments {
		now := time.Now()
		opt := gitlab.CreateMergeRequestDiscussionOptions{
			CreatedAt: &now,
			Body:      &comm.text,
			Position: &gitlab.NotePosition{
				BaseSHA:      diffReffs.BaseSha,
				StartSHA:     diffReffs.StartSha,
				HeadSHA:      diffReffs.HeadSha,
				PositionType: "text",
				NewPath:      comm.file,
				OldPath:      comm.file,
				NewLine:      comm.line,
			},
		}
		json.NewEncoder(os.Stdout).Encode(opt)
		/*
			_, _, err := git.Discussions.CreateMergeRequestDiscussion(3637, 1, &opt)
			if err != nil {
				log.Fatal(err)
			}
		*/
	}
}
