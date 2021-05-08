package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// Commits is struct of commit
type Commits struct {
	Content string
	Count   int
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
	if err != nil {
		return
	}

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
		// 指定された日に積まれたコミットを追加
		if chapter == "git" {
			heading := config.Git.Heading
			if heading == "" {
				heading = chapter
			}
			progress, commitCnt, err := getProgress(cmd, date)
			if err != nil {
				return "", err
			}
			if commitCnt > 0 {
				str = "## " + heading + "\n"
				str += progress + "\n"
			}
		} else if chapter == "slack" {
			// slackのusernameを取得
			username, err := getSlackUserName(cmd)
			if err != nil {
				return "", err
			}

			// slack上での発言を取得
			remark, remarkCnt, err := getRemark(cmd, username, date)
			if err != nil {
				return "", err
			}
			if remarkCnt > 0 {
				str = "## " + chapter + "\n"
				str += remark
			}
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
func getProgress(cmd *cobra.Command, date string) (progress string, commitCnt int, err error) {
	if len(config.Git.Repositories) <= 0 {
		msg := "リポジトリを指定してください\n\n```$HOME/.nippo.yaml\ngit:\n    " +
			"repositories: \n        " +
			"#コミットを取得したいディレクトリの絶対パスを記入してください。\n        " +
			"- \"Users/MasatoraAtarashi/workspace/hogehoge\"\n        " +
			"- \"Users/MasatoraAtarashi/workspace/hogehoge2\"\n```\n"
		return "", 0, errors.New(msg)
	}

	username, err := getGitUserName(cmd)
	if err != nil {
		return
	}

	for _, repository := range config.Git.Repositories {
		var commits Commits
		commits, err = getCommits(repository, username, date)
		if commits.Count > 0 {
			commitCnt += commits.Count
			repositoryName := strings.Split(repository, "/")
			progress += "### " + repositoryName[len(repositoryName)-1] + "(" + strconv.Itoa(commits.Count) + " commits)" + "\n"
			progress += commits.Content
		}
	}
	return
}

// gitのusernameを取得
func getGitUserName(cmd *cobra.Command) (username string, err error) {
	username, err = cmd.PersistentFlags().GetString("gituser")
	if err != nil {
		return
	}

	if username == "" {
		out, err := exec.Command("git", "config", "user.name").Output()
		if err != nil {
			return "", err
		}
		username = string(out)
	}
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

// その日の発言を取得
func getRemark(cmd *cobra.Command, username string, date string) (remark string, remarkCnt int, err error) {
	token := config.Slack.Token
	if token == "" {
		msg := "SlackのAPI Tokenを設定してください\n\n```$HOME/.nippo.yaml\nslack:\n    " +
			"token: \"\" #Slack APIトークンを記入してください。\n    " +
			"username: \"\" #Slackのユーザ名を記入してください。\n```\n"
		return "", 0, errors.New(msg)
	}
	api := slack.New(token)

	const layout = "2006-01-02"
	dateTime, err := time.Parse(layout, date)
	startDate := dateTime.AddDate(0, 0, -1)
	endDate := dateTime.AddDate(0, 0, 1)

	count, err := cmd.PersistentFlags().GetInt8("count")
	if err != nil {
		return
	}
	params := &slack.SearchParameters{
		Sort:          "score",
		SortDirection: "desc",
		Count:         int(count),
	}
	messages, err := api.SearchMessages("from:@"+username+" after:"+startDate.Format(layout)+" before:"+endDate.Format(layout), *params)
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, message := range messages.Matches {
		remark += " - `" + message.Text + "` (" + message.Channel.Name + ")\n"
	}
	remarkCnt = len(messages.Matches)
	return
}

func getSlackUserName(cmd *cobra.Command) (username string, err error) {
	username, err = cmd.PersistentFlags().GetString("slackuser")
	if err != nil {
		return
	}
	if username == "" {
		username = config.Slack.Username
	}
	if username == "" {
		msg := "Slackのユーザー名を設定してください\n\n```$HOME/.nippo.yaml\nslack:\n    " +
			"token: \"\" #Slack APIトークンを記入してください。\n    " +
			"username: \"\" #Slackのユーザ名を記入してください。\n```\n"
		return "", errors.New(msg)
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
	generateCmd.PersistentFlags().StringP("gituser", "g", "", "Specify git username")
	generateCmd.PersistentFlags().StringP("slackuser", "s", "", "Specify slack username")
	generateCmd.PersistentFlags().Int8P("count", "c", 100, "Specify count of remark to get")
	rootCmd.AddCommand(generateCmd)
}
