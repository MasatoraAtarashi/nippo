package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Commits struct {
	Content string
	Count int
}

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
	err = generateNippo(cmd)
	return
}

// 日報作成
func generateNippo(cmd *cobra.Command) (err error) {
	// init default content
	content, err := initContent(cmd)

	// make tmp file
	fpath, err := makeTmpFile(content)
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
	result, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Fprint(os.Stdout, fmt.Sprintf("failed read content. %s\n", err.Error()))
		return
	}
	fmt.Println(string(result))

	return
}

// init content of 日報
func initContent(cmd *cobra.Command) (content string, err error) {
	// configファイルにテンプレートが設定されていなければエラー
	if len(config.Template) <= 0 {
		return "", errors.New("日報のテンプレートを設定してください")
	}

	// 日付を追加
	date, err := getDate(cmd)
	if err != nil {
		return
	}
	content += "# " + date + "\n"

	// コンテンツを追加
	for _, chapter := range config.Template {
		var str string
		if chapter == "git" {
			// 指定された日に積まれたコミットを追加
			str = "## " + chapter + "\n"
			progress, err := getProgress(cmd, date)
			if err != nil {
				return "", err
			}
			str += progress + "\n"
		} else {
			str = "## " + chapter + "\n\n\n"
		}
		content += str
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

// その日の進捗(コミット)を取得
func getProgress(cmd *cobra.Command, date string) (progress string, err error) {
	if len(config.Git.Repositories) <= 0 {
		return "", errors.New("リポジトリを指定してください")
	}

	username, err := getUserName(cmd)
	if err != nil {
		return
	}

	for _, repository := range config.Git.Repositories {
		fmt.Println(repository)
		var commits Commits
		commits, err = getCommits(repository, username, date)
		if commits.Count > 0 {
			repositoryName := strings.Split(repository, "/")
			progress += "### " + repositoryName[len(repositoryName)-1] + "(" + strconv.Itoa(commits.Count) + " commits)" + "\n"
			progress += commits.Content
			progress += "\n"
		}
	}
	return
}

// usernameを取得
func getUserName(cmd *cobra.Command) (username string, err error) {
	//username, err = cmd.PersistentFlags().GetString("user")
	//if err != nil {
	//	return
	//}
	//
	//if username == "" {
	//	out, err := exec.Command("git", "config", "user.name").Output()
	//	if err != nil {
	//		return "", err
	//	}
	//	username = string(out)
	//}
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "", err
	}
	username = string(out)
	return
}

// 指定された日に指定されたリポジトリに指定したユーザが積んだコミットを取得
func getCommits(repository string, username string, date string) (commits Commits, err error) {
	const layout = "2006-01-02"
	startDate, err := time.Parse(layout, date)
	endDate := startDate.AddDate(0, 0, 1)
	cmdArgs := []string{
		"-C", repository, "log",
		"--oneline",
		"--author=" + username,
		"--since=" + startDate.Format(layout) + " 00:00:00",
		"--until=" + endDate.Format(layout) + " 00:00:00",
		"--branches",
		"--format= - %C(auto)%h%Creset %s",
	}

	// gitコマンドを実行
	out, err := execGitCmd(cmdArgs)
	commits.Content = string(out)
	commits.Count = len(strings.Split(commits.Content, "\n")) - 1
	return
}

// gitコマンドを実行
func execGitCmd(cmdArgs []string) (out []byte, err error) {
	cmd := exec.Command(
		"git", cmdArgs...,
	)
	cmd.Stderr = os.Stderr
	out, err = cmd.Output()
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
