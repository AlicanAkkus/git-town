package script

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Originate/git-town/src/exit"
	"github.com/Originate/git-town/src/git"
	"github.com/Originate/git-town/src/util"

	"github.com/fatih/color"
)

// OpenBrowser opens the default browser with the given URL.
func OpenBrowser(url string) {
	command := util.GetOpenBrowserCommand()
	err := RunCommand(command, url)
	exit.On(err)
}

// PrintCommand prints the given command-line operation on the console.
func PrintCommand(cmd ...string) {
	header := ""
	for index, part := range cmd {
		if strings.Contains(part, " ") {
			part = "\"" + strings.Replace(part, "\"", "\\\"", -1) + "\""
		}
		if index != 0 {
			header = header + " "
		}
		header = header + part
	}
	if strings.HasPrefix(header, "git") && git.IsRepository() {
		header = fmt.Sprintf("[%s] %s", git.GetCurrentBranchName(), header)
	}
	fmt.Println()
	_, err := color.New(color.Bold).Println(header)
	exit.On(err)
}

// RunCommand executes the given command-line operation.
func RunCommand(cmd ...string) error {
	PrintCommand(cmd...)
	// Windows commands run inside CMD
	// because opening browsers is done via "start"
	if runtime.GOOS == "windows" {
		cmd = append([]string{"cmd", "/C"}, cmd...)
	}
	subProcess := exec.Command(cmd[0], cmd[1:]...)
	subProcess.Stderr = os.Stderr
	subProcess.Stdin = os.Stdin
	subProcess.Stdout = os.Stdout
	return subProcess.Run()
}

// RunCommandSafe executes the given command-line operation, exiting if the command errors
func RunCommandSafe(cmd ...string) {
	err := RunCommand(cmd...)
	exit.On(err)
}
