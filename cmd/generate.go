package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		err := runGenerateCmd(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
		}
	},
}

func runGenerateCmd(cmd *cobra.Command, args []string) (err error) {
	err = generateNippo()
	return
}

// 日報作成
func generateNippo() (err error) {
	// make tmp file
	fpath, err := makeTmpFile("### nippo\n")
	if err != nil {
		fmt.Fprint(os.Stdout, fmt.Sprintf("failed make edit file. %s\n", err.Error()))
		return
	}
	// delete tmp file
	defer deleteFile(fpath)

	// open text editor
	err = openEditor("vim", fpath)
	if err != nil {
		fmt.Fprint(os.Stdout, fmt.Sprintf("failed open text editor. %s\n", err.Error()))
		return
	}

	// read tmp file
	content, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Fprint(os.Stdout, fmt.Sprintf("failed read content. %s\n", err.Error()))
		return
	}
	fmt.Println(string(content))

	return
}

// make tmp file
func makeTmpFile(msg string) (fpath string, err error) {
	home := os.Getenv("HOME")
	fpath = filepath.Join(home, "NIPPO_EDITMSG")
	if err != nil {
		return
	}
	if !isFileExist(fpath) {
		err = ioutil.WriteFile(fpath, []byte(msg), 0644)
		if err != nil {
			return
		}
	}
	return
}

// ファイルの存在を確認する
func isFileExist(fpath string) bool {
	_, err := os.Stat(fpath)
	return err == nil || !os.IsNotExist(err)
}

// ファイルを削除する
func deleteFile(fpath string) error {
	return os.Remove(fpath)
}

// エディタを開く
func openEditor(program string, fpath string) error {
	c := exec.Command(program, fpath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
