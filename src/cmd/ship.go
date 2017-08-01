package cmd

import (
	"strings"

	"github.com/Originate/git-town/src/git"
	"github.com/Originate/git-town/src/prompt"
	"github.com/Originate/git-town/src/script"
	"github.com/Originate/git-town/src/steps"
	"github.com/Originate/git-town/src/util"

	"github.com/spf13/cobra"
)

type shipConfig struct {
	BranchToShip        string
	InitialBranch       string
	IsTargetBranchLocal bool
}

var commitMessage string

var shipCmd = &cobra.Command{
	Use:   "ship",
	Short: "Deliver a completed feature branch",
	Long: `Deliver a completed feature branch

Squash-merges the current branch, or <branch_name> if given,
into the main branch, resulting in linear history on the main branch.

- syncs the main branch
- pulls remote updates for <branch_name>
- merges the main branch into <branch_name>
- squash-merges <branch_name> into the main branch
  with commit message specified by the user
- pushes the main branch to the remote repository
- deletes <branch_name> from the local and remote repositories

Only shipping of direct children of the main branch is allowed.
To ship a nested child branch, all ancestor branches have to be shipped or killed.`,
	Run: func(cmd *cobra.Command, args []string) {
		git.EnsureIsRepository()
		prompt.EnsureIsConfigured()
		steps.Run(steps.RunOptions{
			CanSkip:              func() bool { return false },
			Command:              "ship",
			IsAbort:              abortFlag,
			IsContinue:           continueFlag,
			IsSkip:               false,
			IsUndo:               undoFlag,
			SkipMessageGenerator: func() string { return "" },
			StepListGenerator: func() steps.StepList {
				config := checkShipPreconditions(args)
				return getShipStepList(config)
			},
		})
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return validateMaxArgs(args, 1)
	},
}

func checkShipPreconditions(args []string) (result shipConfig) {
	result.InitialBranch = git.GetCurrentBranchName()
	if len(args) == 0 {
		result.BranchToShip = result.InitialBranch
	} else {
		result.BranchToShip = args[0]
	}
	if result.BranchToShip == result.InitialBranch {
		git.EnsureDoesNotHaveOpenChanges("Did you mean to commit them before shipping?")
	}
	if git.HasRemote("origin") && !git.IsOffline() {
		script.Fetch()
	}
	if result.BranchToShip != result.InitialBranch {
		git.EnsureHasBranch(result.BranchToShip)
	}
	git.EnsureIsFeatureBranch(result.BranchToShip, "Only feature branches can be shipped.")
	prompt.EnsureKnowsParentBranches([]string{result.BranchToShip})
	ensureParentBranchIsMainBranch(result.BranchToShip)
	return
}

func ensureParentBranchIsMainBranch(branchName string) {
	if git.GetParentBranch(branchName) != git.GetMainBranch() {
		ancestors := git.GetAncestorBranches(branchName)
		ancestorsWithoutMain := ancestors[1:]
		oldestAncestor := ancestorsWithoutMain[0]
		util.ExitWithErrorMessage(
			"Shipping this branch would ship "+strings.Join(ancestorsWithoutMain, ", ")+" as well.",
			"Please ship \""+oldestAncestor+"\" first.",
		)
	}
}

func getShipStepList(config shipConfig) (result steps.StepList) {
	var isOffline = git.IsOffline()
	mainBranch := git.GetMainBranch()
	isShippingInitialBranch := config.BranchToShip == config.InitialBranch
	result.AppendList(steps.GetSyncBranchSteps(mainBranch))
	result.Append(steps.CheckoutBranchStep{BranchName: config.BranchToShip})
	result.Append(steps.MergeTrackingBranchStep{})
	result.Append(steps.MergeBranchStep{BranchName: mainBranch})
	result.Append(steps.EnsureHasShippableChangesStep{BranchName: config.BranchToShip})
	result.Append(steps.CheckoutBranchStep{BranchName: mainBranch})
	result.Append(steps.SquashMergeBranchStep{BranchName: config.BranchToShip, CommitMessage: commitMessage})
	if git.HasRemote("origin") && !isOffline {
		result.Append(steps.PushBranchStep{BranchName: mainBranch, Undoable: true})
	}
	childBranches := git.GetChildBranches(config.TargetBranch)
	if git.HasTrackingBranch(config.TargetBranch) && len(childBranches) == 0 && !isOffline {
		result.Append(steps.DeleteRemoteBranchStep{BranchName: config.BranchToShip, IsTracking: true})
	}
	result.Append(steps.DeleteLocalBranchStep{BranchName: config.BranchToShip})
	result.Append(steps.DeleteParentBranchStep{BranchName: config.BranchToShip})
	for _, child := range childBranches {
		result.Append(steps.SetParentBranchStep{BranchName: child, ParentBranchName: mainBranch})
	}
	result.Append(steps.DeleteAncestorBranchesStep{})
	if !isShippingInitialBranch {
		result.Append(steps.CheckoutBranchStep{BranchName: config.InitialBranch})
	}
	result.Wrap(steps.WrapOptions{RunInGitRoot: true, StashOpenChanges: !isShippingInitialBranch})
	return
}

func init() {
	shipCmd.Flags().BoolVar(&abortFlag, "abort", false, abortFlagDescription)
	shipCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Specify the commit message for the squash commit")
	shipCmd.Flags().BoolVar(&continueFlag, "continue", false, continueFlagDescription)
	shipCmd.Flags().BoolVar(&undoFlag, "undo", false, undoFlagDescription)
	RootCmd.AddCommand(shipCmd)
}
