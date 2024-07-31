package gitsync

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/nieomylnieja/gitsync/internal/config"
	"github.com/nieomylnieja/gitsync/internal/diff"
)

type Command int

const (
	CommandSync Command = iota
	CommandDiff
)

const (
	gitsyncUpdateBranch = "gitsync-update"
	commitBaseMessage   = "chore: gitsync update"
	promptMessage       = "Accept hunk? [Y|y|n|i|h]: "
	gitsyncURL          = "https://github.com/nieomylnieja/gitsync"
)

func Run(conf *config.Config, command Command) error {
	if err := checkDependencies(); err != nil {
		return err
	}
	// #nosec G304
	if err := os.MkdirAll(conf.GetStorePath(), 0o750); err != nil {
		return errors.Wrap(err, "failed to create repositories store under specified path")
	}
	for _, repo := range append(conf.Repositories, conf.Root) {
		if err := cloneRepo(repo); err != nil {
			return errors.Wrapf(err, "failed to clone repository %s", repo.Name)
		}
		if err := updateTrackedRef(repo); err != nil {
			return errors.Wrapf(err, "failed to update repository %s", repo.Name)
		}
		if command == CommandSync {
			if err := checkoutSyncBranch(repo); err != nil {
				return err
			}
		}
	}
	updatedFiles := make(map[*config.Repository][]string, len(conf.Repositories))
	for _, syncedRepo := range conf.Repositories {
		for _, file := range conf.SyncFiles {
			rootFilePath := filepath.Join(conf.GetStorePath(), conf.Root.Name, file.Path)
			updated, err := syncRepoFile(conf, command, syncedRepo, file, rootFilePath)
			if err != nil {
				return errors.Wrapf(err, "failed to sync %s repository file: %s", syncedRepo.Name, file.Name)
			}
			if updated {
				updatedFiles[syncedRepo] = append(updatedFiles[syncedRepo], file.Path)
			}
		}
	}
	if command == CommandDiff {
		return nil
	}
	if len(updatedFiles) == 0 {
		fmt.Println("No changes to synchronize.")
		return nil
	}
	for repo, files := range updatedFiles {
		commit, err := commitChanges(conf.Root, repo, files)
		if err != nil {
			return errors.Wrapf(err, "failed to commit changes to %s repository", repo.Name)
		}
		if err = pushChanges(repo); err != nil {
			return errors.Wrapf(err, "failed to push changes to %s repository", repo.Name)
		}
		if err = openPullRequest(repo, commit); err != nil {
			return errors.Wrapf(err, "failed to open pull request for %s repository", repo.Name)
		}
	}
	return nil
}

func syncRepoFile(
	conf *config.Config,
	command Command,
	syncedRepo *config.Repository,
	file *config.File,
	rootFilePath string,
) (bool, error) {
	syncedRepoFilePath := filepath.Join(conf.GetStorePath(), syncedRepo.Name, file.Path)
	regexes := make([]string, 0)
	for _, ignore := range getIgnoreRules(conf, ignoreRulesQuery{
		RepoName: syncedRepo.Name,
		FileName: file.Name,
		Regex:    true,
	}) {
		regexes = append(regexes, *ignore.Regex)
	}
	args := []string{
		"-U", "0",
		"--color=always",
		"--label", fmt.Sprintf("%s (synced): %s (%s)", syncedRepo.Name, file.Path, file.Name),
		"--label", fmt.Sprintf("%s (root): %s (%s)", conf.Root.Name, file.Path, file.Name),
	}
	for _, regex := range regexes {
		args = append(args, "-I")
		args = append(args, regex)
	}
	args = append(args,
		syncedRepoFilePath,
		rootFilePath)
	out, err := newCmd().
		SkipErroneousStatus(1).
		Exec("diff", args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to execute diff command")
	}
	if out.Len() == 0 {
		return false, nil
	}
	unifiedFmt, err := diff.ParseDiffOutput(out)
	if err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(os.Stdin)
	resultHunks := make([]diff.Hunk, 0, len(unifiedFmt.Hunks))
	prompt := command == CommandSync
hunkLoop:
	for _, hunk := range unifiedFmt.Hunks {
		for _, ignore := range getIgnoreRules(conf, ignoreRulesQuery{
			RepoName: syncedRepo.Name,
			FileName: file.Name,
			Hunk:     true,
		}) {
			if ignore.Hunk.Equal(hunk) {
				continue hunkLoop
			}
		}
		if !prompt {
			resultHunks = append(resultHunks, hunk)
			continue
		}

		sep := getPrintSeparator(append(hunk.Changes, strings.Split(unifiedFmt.Header, "\n")...))
		fmt.Printf("%[1]s\n%[2]s\n%[3]s%[1]s\n", sep, unifiedFmt.Header, hunk.Original)
		fmt.Print(promptMessage)
		for scanner.Scan() {
			switch scanner.Text() {
			case "Y":
				resultHunks = append(resultHunks, hunk)
				prompt = false
			case "y", "yes":
				resultHunks = append(resultHunks, hunk)
			case "n", "no":
			case "i":
				// Copy loop variable.
				hunk := hunk
				conf.Ignore = append(conf.Ignore, &config.IgnoreRule{
					RepositoryName: &syncedRepo.Name,
					FileName:       &file.Name,
					Hunk:           &hunk,
				})
			default:
				fmt.Println("Invalid input. Please enter Y (all), y (yes), n (no), i (ignore), or h (help).")
				fmt.Print(promptMessage)
				continue
			}
			break
		}
	}
	unifiedFmt.Hunks = resultHunks
	if len(unifiedFmt.Hunks) == 0 {
		return false, nil
	}
	switch command {
	case CommandDiff:
		patch := unifiedFmt.String(true)
		sep := getPrintSeparator(strings.Split(patch, "\n"))
		fmt.Printf("%s\n%s", sep, patch)
		return false, nil
	case CommandSync:
		patch := unifiedFmt.String(false)
		if err = applyPatch(syncedRepoFilePath, patch); err != nil {
			return false, err
		}
	}
	return true, nil
}

func applyPatch(path, patch string) error {
	fmt.Printf("Applying patch to %s\n", path)
	if _, err := newCmd().
		SetStdin(bytes.NewBufferString(patch)).
		Exec(
			"patch",
			path,
			"--input=-",
			"--reject-file=-",
			"--silent",
			"--unified",
			"--force",
		); err != nil {
		fmt.Printf("Patch:\n%s\n", patch)
		return errors.Wrap(err, "failed to apply patch")
	}
	return nil
}

type commitDetails struct {
	Title string
	Body  string
}

func commitChanges(root, repo *config.Repository, changedFiles []string) (*commitDetails, error) {
	path := repo.GetPath()
	fmt.Printf("%s: adding changes to the index\n", repo.Name)
	if _, err := execCmd(
		"git",
		"-C", path,
		"add",
		"--all",
	); err != nil {
		return nil, errors.Wrap(err, "failed to add changes to the index")
	}
	fmt.Printf("%s: committing changes\n", repo.Name)
	message := commitBaseMessage
	var body strings.Builder
	body.WriteString("Synced the following files:\n\n")
	for _, file := range changedFiles {
		body.WriteString(fmt.Sprintf("- %s\n", file))
	}
	body.WriteString(fmt.Sprintf("\nRoot repository ref: %s\n", strings.TrimSuffix(root.URL, ".git")))
	bodyStr := body.String()
	if _, err := execCmd(
		"git",
		"-C", path,
		"commit",
		"-m", message,
		"-m", bodyStr,
	); err != nil {
		return nil, errors.Wrap(err, "failed to commit changes")
	}
	return &commitDetails{
		Title: message,
		Body:  bodyStr,
	}, nil
}

func pushChanges(repo *config.Repository) error {
	path := repo.GetPath()
	fmt.Printf("%s: pushing changes to remote\n", repo.Name)
	if _, err := execCmd(
		"git",
		"-C", path,
		"push",
		"--force",
		"-u",
		"origin",
		gitsyncUpdateBranch,
	); err != nil {
		return errors.Wrap(err, "failed to push changes to remote")
	}
	return nil
}

type ghPullRequest struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func openPullRequest(repo *config.Repository, commit *commitDetails) error {
	ref := repo.GetRef()
	u, err := url.Parse(repo.URL)
	if err != nil {
		return errors.Wrap(err, "failed to parse repository URL")
	}
	ghRepo := filepath.Join(u.Host, strings.TrimSuffix(u.Path, ".git"))
	out, err := execCmd("gh", "auth", "token")
	if err != nil {
		return errors.Wrap(err, "failed to get GitHub token")
	}
	ghToken := strings.TrimSpace(out.String())
	out, err = newCmd().
		WithEnv("GH_TOKEN", ghToken).
		Exec(
			"gh",
			"-R", ghRepo,
			"pr",
			"list",
			"--search", commitBaseMessage,
			"--json", "title,url",
		)
	if err != nil {
		return errors.Wrap(err, "failed to list GitHub pull requests")
	}
	var prs []ghPullRequest
	if err = json.Unmarshal(out.Bytes(), &prs); err != nil {
		return errors.Wrap(err, "failed to unmarshal GitHub pull requests list response")
	}
	if len(prs) > 0 {
		for _, pr := range prs {
			if pr.Title == commit.Title {
				fmt.Printf("%s: pull request already exists, skipping creation (%s)\n", repo.Name, pr.URL)
				return nil
			}
		}
	}
	fmt.Printf("%s: opening GitHub pull request\n", repo.Name)
	body := commit.Body
	body += fmt.Sprintf("\nPull request generated by [gitsync](%s)", gitsyncURL)
	out, err = newCmd().
		WithEnv("GH_TOKEN", ghToken).
		Exec(
			"gh",
			"-R", ghRepo,
			"pr",
			"create",
			"--title", commit.Title,
			"--body", body,
			"--assignee", "@me",
			// It's vital to remove the "origin/" prefix.
			// Otherwise, the GitHub CLI will fail to create a pull request,
			// as it can only accept a direct branch name.
			"--base", strings.TrimPrefix(ref, "origin/"),
			"--head", gitsyncUpdateBranch,
		)
	if err != nil {
		return errors.Wrap(err, "failed to push changes to remote")
	}
	prURL := strings.TrimSpace(out.String())
	fmt.Printf("%s: pull request URL: %s\n", repo.Name, prURL)
	return nil
}

func cloneRepo(repo *config.Repository) error {
	path := repo.GetPath()
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil
	}
	fmt.Printf("%s: cloning %s into %s\n", repo.Name, repo.URL, path)
	if _, err := execCmd(
		"git",
		"clone",
		"--",
		repo.URL,
		path,
	); err != nil {
		return errors.Wrap(err, "failed to clone repository")
	}
	return nil
}

func updateTrackedRef(repo *config.Repository) error {
	path := repo.GetPath()
	ref := repo.GetRef()
	fmt.Printf("%s: updating repository ref (%s)\n", repo.Name, ref)
	if _, err := execCmd(
		"git",
		"-C", path,
		"fetch",
		"--force",
		"--all",
	); err != nil {
		return errors.Wrap(err, "failed to fetch repository objects and refs")
	}
	if _, err := execCmd(
		"git",
		"-C", path,
		"checkout",
		"--force",
		ref,
	); err != nil {
		return errors.Wrapf(err, "failed to force checkout %s ref", ref)
	}
	if _, err := execCmd(
		"git",
		"-C", path,
		"reset",
		"--hard",
		ref,
	); err != nil {
		return errors.Wrapf(err, "failed to hard reset repository to %s ref", ref)
	}
	return nil
}

func checkoutSyncBranch(repo *config.Repository) error {
	path := repo.GetPath()
	ref := repo.GetRef()
	fmt.Printf("%s: checking out %s branch\n", repo.Name, gitsyncUpdateBranch)
	if _, err := execCmd(
		"git",
		"-C", path,
		"checkout",
		"--force",
		"-B",
		gitsyncUpdateBranch,
		ref,
	); err != nil {
		return errors.Wrap(err, "failed to create/reset gitsync branch")
	}
	return nil
}

func getPrintSeparator(strs []string) string {
	maxLineLen := len(slices.MaxFunc(
		strs,
		func(a, b string) int { return cmp.Compare(len(a), len(b)) },
	))
	return strings.Repeat("=", maxLineLen)
}

func checkDependencies() error {
	if _, err := execCmd("git", "--version"); err != nil {
		return errors.New("'git' is required to be installed")
	}
	if _, err := execCmd("gh", "--version"); err != nil {
		return errors.New("'gh' (GitHub CLI) is required to be installed")
	}
	if _, err := execCmd("diff", "--version"); err != nil {
		return errors.New("'diff' (GNU) is required to be installed")
	}
	return nil
}

type ignoreRulesQuery struct {
	RepoName string
	FileName string
	Hunk     bool
	Regex    bool
}

func getIgnoreRules(conf *config.Config, query ignoreRulesQuery) []*config.IgnoreRule {
	if len(conf.Ignore) == 0 {
		return nil
	}
	rules := make([]*config.IgnoreRule, 0)
	for _, ignore := range conf.Ignore {
		if ignore.RepositoryName != nil && *ignore.RepositoryName != query.RepoName {
			continue
		}
		if ignore.FileName != nil && *ignore.FileName != query.FileName {
			continue
		}
		if query.Hunk && ignore.Hunk != nil {
			rules = append(rules, ignore)
		}
		if query.Regex && ignore.Regex != nil {
			rules = append(rules, ignore)
		}
	}
	return rules
}
