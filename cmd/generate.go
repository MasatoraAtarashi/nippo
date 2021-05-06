package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate nippo",
	Run: func(cmd *cobra.Command, args []string) {
		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := viper.Unmarshal(&config); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		err := runGenerateCmd(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
		}
	},
}

func runGenerateCmd(cmd *cobra.Command, args []string) (err error) {
	fmt.Println(config.Template)
	err = generateNippo(cmd)
	return
}

// 日報作成
func generateNippo(cmd *cobra.Command) (err error) {
	// init default content
	defaultContent, err := initDefaultContent(cmd)

	// make tmp file
	fpath, err := makeTmpFile(defaultContent)
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

// init default content of 日報
func initDefaultContent(cmd *cobra.Command) (defaultContent string, err error) {
	if len(config.Template) <= 0 {
		return "", errors.New("日報のテンプレートを設定してください")
	}

	// 日付を取得
	date, err := getDate(cmd)
	if err != nil {
		return
	}

	defaultContent += "# " + date + "\n"

	for _, chapter := range config.Template {
		str := "## " + chapter + "\n\n\n"
		defaultContent += str
	}
	return
}

// 日付を取得
func getDate(cmd *cobra.Command) (date string, err error) {
	const layout = "2006-01-02"
	date, err = cmd.PersistentFlags().GetString("date")
	if err != nil {
		return
	}
	if date == "" {
		date = time.Now().Format(layout)
	}
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
	generateCmd.PersistentFlags().StringP("date", "d", "", "Specify date like <2021-04-24>")
	rootCmd.AddCommand(generateCmd)
}
