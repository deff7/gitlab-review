package main

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
)

type view struct {
	client        *gitlabClient
	filesComments []fileComments
	curFileIdx    int
	curCommentIdx int
	scroll        int
}

func newView(client *gitlabClient, filesComments []fileComments) error {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	vw := view{
		filesComments: filesComments,
		client:        client,
	}
	g.SetManagerFunc(vw.layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return errors.Wrap(err, "set keybinding")
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return errors.Wrap(err, "set keybinding")
	}
	if err := g.SetKeybinding("", 'n', gocui.ModNone, vw.skip); err != nil {
		return errors.Wrap(err, "set keybinding 'n'")
	}
	if err := g.SetKeybinding("", 'y', gocui.ModNone, vw.push); err != nil {
		return errors.Wrap(err, "set keybinding 'y'")
	}
	if err := g.SetKeybinding("", 'j', gocui.ModNone, vw.scrollDown); err != nil {
		return errors.Wrap(err, "set keybinding 'y'")
	}
	if err := g.SetKeybinding("", 'k', gocui.ModNone, vw.scrollUp); err != nil {
		return errors.Wrap(err, "set keybinding 'y'")
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		return errors.Wrap(err, "main loop")
	}
	return nil
}

func fprintColor(w io.Writer, s string, color int) {
	for _, line := range strings.Split(s, "\n") {
		fmt.Fprintf(w, "\033[3%d;%dm%s\033[0m\n", color, color, line)
	}
}

// CR: foo?
// ```go
// func foo() {
//   log.Fatal("lol")
// }
// ```
func (vw view) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("main", 0, 0, maxX-1, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Commentary"
		vw.drawComment(v)
	}

	if v, err := g.SetView("file", 0, maxY/2, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		vw.drawFile(v)
	}
	return nil
}

func (vw view) currentComment() comment {
	return vw.currentFile().comments[vw.curCommentIdx]
}

func (vw view) currentFile() fileComments {
	return vw.filesComments[vw.curFileIdx]
}

func (vw view) drawComment(v *gocui.View) error {
	v.Clear()

	comm := vw.currentComment()
	file := vw.currentFile()

	fmt.Fprintln(v, "Push this comment? [y/n]\n")
	fprintColor(v, fmt.Sprintf("File: %s:%d\n", file.fileName, comm.line), 1)
	fmt.Fprintln(v, comm.text)
	return nil
}

func (vw view) drawFile(v *gocui.View) error {
	v.Clear()
	v.SetOrigin(0, vw.scroll)
	file := vw.currentFile()
	v.Title = file.fileName + "─Use j/k/Up/Down for scrolling"
	drawFile(v, file, vw.curCommentIdx)
	//fmt.Fprintln(v, file.fileBody)
	return nil
}

func (vw *view) push(g *gocui.Gui, v *gocui.View) error {
	v, err := g.View("main")
	if err != nil {
		return errors.Wrap(err, "get view")
	}
	fmt.Fprintln(v, "Pushing...")
	err = vw.client.pushComment(vw.currentFile().fileName, vw.currentComment())
	if err != nil {
		return errors.Wrap(err, "push comment")
	}
	if err := vw.nextComment(); err != nil {
		return err
	}
	g.Update(vw.update)

	return nil
}

func (vw *view) nextComment() error {
	vw.curCommentIdx++
	if vw.curCommentIdx == len(vw.currentFile().comments) {
		vw.curFileIdx++
		vw.curCommentIdx = 0
		if vw.curFileIdx == len(vw.filesComments) {
			return gocui.ErrQuit
		}
	}
	vw.scroll = vw.currentComment().line - 1
	return nil
}

func (vw *view) skip(g *gocui.Gui, v *gocui.View) error {
	if err := vw.nextComment(); err != nil {
		return err
	}
	g.Update(vw.update)
	return nil
}

func (vw *view) scrollDown(g *gocui.Gui, v *gocui.View) error {
	v, err := g.View("file")
	if err != nil {
		return errors.Wrap(err, "get view")
	}
	vw.scroll++
	g.Update(vw.updateFileView)

	return nil
}

func (vw *view) scrollUp(g *gocui.Gui, v *gocui.View) error {
	v, err := g.View("file")
	if err != nil {
		return errors.Wrap(err, "get view")
	}
	if vw.scroll > 0 {
		vw.scroll--
	}
	g.Update(vw.updateFileView)

	return nil
}

func (vw view) update(g *gocui.Gui) error {
	v, err := g.View("main")
	if err != nil {
		return err
	}
	vw.drawComment(v)

	v, err = g.View("file")
	if err != nil {
		return err
	}
	vw.drawFile(v)
	return nil
}

func (vw view) updateFileView(g *gocui.Gui) error {
	v, err := g.View("file")
	if err != nil {
		return err
	}
	vw.drawFile(v)
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func drawFile(w io.Writer, f fileComments, curCommentIdx int) {
	lines := strings.Split(f.fileBody, "\n")
	comments := f.comments
	curComment := comments[curCommentIdx]

	filtered := lines[:0]
	for i, line := range lines {
		lineNum := i + 1

		if len(comments) == 0 {
			filtered = append(filtered, line)
			continue
		}

		comm := comments[0]
		if lineNum < comm.start {
			filtered = append(filtered, line)
			continue
		}
		if lineNum > comm.end {
			comments = comments[1:]
			filtered = append(filtered, line)
			continue
		}
	}

	for i, line := range filtered {
		lineNum := i + 1
		line = fmt.Sprintf("%3d %s", lineNum, line)
		if lineNum == curComment.line {
			fprintColor(w, line, 2)
			continue
		}
		fmt.Fprintln(w, line)
	}
}
