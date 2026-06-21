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

const REPO_URL = "https://github.com/XboxDev/extract-xiso.git"

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

	x.log("\n$ " + strings.Join(cmd, " ") + "\n\n")

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

	return command.Wait()
}

func (x *XboxExtractor) extractISO(isoPath string) error {

	x.log("\n=== PREPARING ===\n")

	isoDir := filepath.Dir(isoPath)

	isoName := strings.TrimSuffix(
		filepath.Base(isoPath),
		filepath.Ext(isoPath),
	)

	extractedFolder := filepath.Join(
		isoDir,
		isoName+" [Extracted]",
	)

	repoDir := filepath.Join(
		extractedFolder,
		"extract-xiso",
	)

	buildDir := filepath.Join(
		repoDir,
		"build",
	)

	if err := os.MkdirAll(extractedFolder, 0755); err != nil {
		return err
	}

	x.log("\nOutput Folder:\n" + extractedFolder + "\n")

	x.log("\n=== CLONING extract-xiso ===\n")

	if err := x.runCommand(
		[]string{
			"git",
			"clone",
			REPO_URL,
			repoDir,
		},
		"",
	); err != nil {
		return err
	}

	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return err
	}

	x.log("\n=== BUILDING ===\n")

	if err := x.runCommand(
		[]string{
			"cmake",
			"..",
		},
		buildDir,
	); err != nil {
		return err
	}

	if err := x.runCommand(
		[]string{
			"make",
		},
		buildDir,
	); err != nil {
		return err
	}

	binary := filepath.Join(
		buildDir,
		"extract-xiso",
	)

	if _, err := os.Stat(binary); err != nil {
		return fmt.Errorf(
			"extract-xiso binary not found after build",
		)
	}

	x.log("\n=== EXTRACTING ISO ===\n")

	if err := x.runCommand(
		[]string{
			binary,
			"-x",
			isoPath,
		},
		extractedFolder,
	); err != nil {
		return err
	}

	x.log("\n=== CLEANING UP ===\n")

	if err := os.RemoveAll(repoDir); err != nil {
		x.log(
			fmt.Sprintf(
				"Could not remove repo folder: %v\n",
				err,
			),
		)
	}

	x.log("\n=== DONE ===\n")

	exec.Command(
		"xdg-open",
		extractedFolder,
	).Start()

	return nil
}

func (x *XboxExtractor) worker() {

	isoPath := strings.TrimSpace(
		x.isoEntry.Text,
	)

	if isoPath == "" {
		dialog.ShowError(
			fmt.Errorf("please select an ISO first"),
			x.window,
		)
		return
	}

	if err := x.extractISO(isoPath); err != nil {

		x.log(
			"\n\nERROR:\n" +
				err.Error() +
				"\n",
		)

		dialog.ShowError(
			err,
			x.window,
		)

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

	w := a.NewWindow(
		"Xbox ISO Extractor",
	)

	w.Resize(
		fyne.NewSize(
			900,
			600,
		),
	)

	isoEntry := widget.NewEntry()

	logOutput := widget.NewMultiLineEntry()
	logOutput.Disable()

	app := &XboxExtractor{
		window:    w,
		isoEntry:  isoEntry,
		logOutput: logOutput,
	}

	browseBtn := widget.NewButton(
		"Browse ISO",
		app.browseISO,
	)

	extractBtn := widget.NewButton(
		"Install + Extract",
		func() {
			go app.worker()
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
		container.NewScroll(logOutput),
	)

	w.SetContent(content)

	w.ShowAndRun()
}
