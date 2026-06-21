package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	REPO_URL = "https://github.com/XboxDev/extract-xiso.git"
	REPO_DIR = "extract-xiso"
)

type XboxExtractor struct {
	window    fyne.Window
	isoEntry  *widget.Entry
	logOutput *widget.Entry
	mutex     sync.Mutex
}

func (x *XboxExtractor) log(text string) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	fyne.Do(func() {
		x.logOutput.SetText(x.logOutput.Text + text)
	})
}

func (x *XboxExtractor) browseISO() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, x.window)
			return
		}

		if reader == nil {
			return
		}

		x.isoEntry.SetText(reader.URI().Path())
		_ = reader.Close()

	}, x.window)
}

func (x *XboxExtractor) runCommand(cmd []string, cwd string) error {

	x.log(fmt.Sprintf("\n$ %s\n\n", strings.Join(cmd, " ")))

	command := exec.Command(cmd[0], cmd[1:]...)

	if cwd != "" {
		command.Dir = cwd
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}

	command.Stderr = command.Stdout

	if err := command.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stdout)

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			x.log(line)
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	if err := command.Wait(); err != nil {
		return err
	}

	return nil
}

func (x *XboxExtractor) installExtractXiso() (string, error) {

	binary := filepath.Join(
		REPO_DIR,
		"build",
		"extract-xiso",
	)

	if _, err := os.Stat(binary); err == nil {
		x.log("\nextract-xiso already installed.\n")
		return filepath.Abs(binary)
	}

	x.log("\n=== INSTALLING extract-xiso ===\n")

	if _, err := os.Stat(REPO_DIR); os.IsNotExist(err) {

		err = x.runCommand([]string{
			"git",
			"clone",
			REPO_URL,
		}, "")

		if err != nil {
			return "", err
		}
	}

	buildDir := filepath.Join(REPO_DIR, "build")

	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", err
	}

	if err := x.runCommand(
		[]string{"cmake", ".."},
		buildDir,
	); err != nil {
		return "", err
	}

	if err := x.runCommand(
		[]string{"make"},
		buildDir,
	); err != nil {
		return "", err
	}

	if _, err := os.Stat(binary); err != nil {
		return "", fmt.Errorf(
			"build completed but extract-xiso was not found",
		)
	}

	return filepath.Abs(binary)
}

func (x *XboxExtractor) extractISO(binary string, isoPath string) error {

	x.log("\n=== EXTRACTING ISO ===\n")

	isoDir := filepath.Dir(isoPath)

	isoName := strings.TrimSuffix(
		filepath.Base(isoPath),
		filepath.Ext(isoPath),
	)

	extractedFolder := filepath.Join(
		isoDir,
		isoName+" [Extracted]",
	)

	if err := os.MkdirAll(
		extractedFolder,
		0755,
	); err != nil {
		return err
	}

	x.log(
		fmt.Sprintf(
			"\nOutput folder:\n%s\n",
			extractedFolder,
		),
	)

	before := map[string]bool{}

	files, _ := os.ReadDir(isoDir)

	for _, f := range files {
		before[f.Name()] = true
	}

	if err := x.runCommand(
		[]string{
			binary,
			"-x",
			isoPath,
		},
		isoDir,
	); err != nil {
		return err
	}

	after, _ := os.ReadDir(isoDir)

	for _, f := range after {

		if before[f.Name()] {
			continue
		}

		src := filepath.Join(
			isoDir,
			f.Name(),
		)

		if src == extractedFolder {
			continue
		}

		dst := filepath.Join(
			extractedFolder,
			f.Name(),
		)

		if err := os.Rename(src, dst); err != nil {
			x.log(
				fmt.Sprintf(
					"Could not move %s: %v\n",
					f.Name(),
					err,
				),
			)
		}
	}

	x.log("\n=== DONE ===\n")

	exec.Command(
		"xdg-open",
		extractedFolder,
	).Start()

	return nil
}

func (x *XboxExtractor) worker() {

	isoPath := x.isoEntry.Text

	if isoPath == "" {
		dialog.ShowError(
			fmt.Errorf("please select an ISO first"),
			x.window,
		)
		return
	}

	binary, err := x.installExtractXiso()

	if err != nil {
		dialog.ShowError(err, x.window)
		return
	}

	err = x.extractISO(
		binary,
		isoPath,
	)

	if err != nil {
		dialog.ShowError(err, x.window)
		return
	}

	dialog.ShowInformation(
		"Finished",
		"Extraction completed!",
		x.window,
	)
}

func main() {

	a := app.New()

	w := a.NewWindow("Xbox ISO Extractor")
	w.Resize(fyne.NewSize(900, 600))

	isoEntry := widget.NewEntry()

	logOutput := widget.NewMultiLineEntry()
	logOutput.Disable()

	extractor := &XboxExtractor{
		window:    w,
		isoEntry:  isoEntry,
		logOutput: logOutput,
	}

	browseBtn := widget.NewButton(
		"Browse ISO",
		extractor.browseISO,
	)

	extractBtn := widget.NewButton(
		"Install + Extract",
		func() {
			go extractor.worker()
		},
	)

	top := container.NewBorder(
		nil,
		nil,
		nil,
		browseBtn,
		isoEntry,
	)

	content := container.NewBorder(
		container.NewVBox(
			top,
			extractBtn,
		),
		nil,
		nil,
		nil,
		logOutput,
	)

	w.SetContent(content)

	w.ShowAndRun()
}
