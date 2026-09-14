package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kursku/vercelgate/gen/ent"
	"github.com/kursku/vercelgate/gen/ent/team"
	"github.com/kursku/vercelgate/pkg/constants"
	"github.com/kursku/vercelgate/pkg/entcfn"
	"github.com/kursku/vercelgate/pkg/entdb"
	"github.com/kursku/vercelgate/pkg/utils"
	"github.com/kursku/vercelgate/pkg/vercelfn"
	"github.com/kursku/vercelgate/pkg/vercelutil"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

var version = "dev"

func main() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(switchTeamCmd)
	rootCmd.AddCommand(pathCmd)
	rootCmd.AddCommand(accountsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "vercelgate",
	Version: version,
	Short:   "Make vercel cli more powerful by adding the ability to switch between multiple accounts.",
	Long:    `You can swithc between multiple accounts without having relogin and logout.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Run command `vercelgate --help` for more information`")
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run this command very first time",
	Run: func(cmd *cobra.Command, args []string) {
		globalPath, err := vercelutil.GetGlobalPathConfig()
		if err != nil {
			log.Fatal(err)

			return
		}

		err = utils.IsFileExists(filepath.Join(globalPath, constants.DB_FILE_NAME))
		if err == nil {
			fmt.Println("was initialized already")
			return
		}

		err = entcfn.Migrate()
		if err != nil {
			log.Fatal(err)

			return
		}
		fmt.Println("vercelgate initialized successfully")
	},
}

var switchCmd = &cobra.Command{
	Use:   "switch [account]",
	Short: "Switch between account. Pass a name/email/username to switch non-interactively.",
	Run: func(cmd *cobra.Command, args []string) {
		err := SwitchCmd(false, args)

		if err != nil {
			log.Fatal(err)

			return
		}
	},
}

var switchTeamCmd = &cobra.Command{
	Use:   "switchteam [account]",
	Short: "Switch between account and teams",
	Run: func(cmd *cobra.Command, args []string) {
		err := SwitchCmd(true, args)

		if err != nil {
			log.Fatal(err)

			return
		}
	},
}

func SwitchCmd(switchTeam bool, args []string) error {
	var user *ent.User
	var err error
	if len(args) > 0 {
		user, err = findUser(args[0])
	} else {
		user, err = promptGetUser()
	}
	if err != nil {
		return err
	}

	err = vercelutil.SetAuthToken(user.Token)
	if err != nil {
		return err
	}

	displayName := user.Name
	if len(displayName) == 0 {
		displayName = user.Username
	}
	fmt.Printf("Switched to user %s\n", displayName)

	if switchTeam {
		team, err := promptGetTeam(user.ID)
		if err != nil {
			return err
		}

		err = vercelutil.SetCurrentTeam(team.ID)
		if err != nil {
			return err
		}

		fmt.Printf("Switched to team %s\n", team.Name)
	} else {
		vercelutil.DeleteCurrentTeam()
	}

	return nil
}

func promptGetTeam(userID string) (*ent.Team, error) {
	ctx := context.Background()

	items, err := entdb.Client().Team.Query().Where(team.UserID(userID)).All(ctx)

	if err != nil {
		return nil, err
	}

	itemsList := []string{}

	for _, user := range items {
		itemsList = append(itemsList, user.Name)
	}

	prompt := promptui.Select{
		Label: "Select Team",
		Items: itemsList,
	}

	index, _, err := prompt.Run()

	if err != nil {
		return nil, fmt.Errorf("Prompt failed %v\n", err)
	}

	return items[index], nil
}

func findUser(q string) (*ent.User, error) {
	ctx := context.Background()

	users, err := entdb.Client().User.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	ql := strings.ToLower(q)
	var matches []*ent.User
	for _, u := range users {
		if strings.ToLower(u.Name) == ql || strings.ToLower(u.Email) == ql || strings.ToLower(u.Username) == ql {
			return u, nil
		}
		if strings.Contains(strings.ToLower(u.Name), ql) ||
			strings.Contains(strings.ToLower(u.Email), ql) ||
			strings.Contains(strings.ToLower(u.Username), ql) {
			matches = append(matches, u)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, fmt.Errorf("no account matches %q", q)
	default:
		return nil, fmt.Errorf("multiple accounts match %q; be more specific", q)
	}
}

func promptGetUser() (*ent.User, error) {
	ctx := context.Background()

	users, err := entdb.Client().User.Query().All(ctx)

	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("no accounts synced yet. please sync first using `vercelgate sync`")
	}

	usersList := []string{}

	for _, user := range users {
		name := user.Name
		if len(name) == 0 {
			name = user.Username
		}
		usersList = append(usersList, fmt.Sprintf("%s (%s)", name, user.Email))
	}

	prompt := promptui.Select{
		Label: "Select Account",
		Items: usersList,
	}

	index, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return nil, err
	}

	return users[index], nil
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current logged in account",
	Run: func(cmd *cobra.Command, args []string) {
		err := vercelfn.SyncAuthJson()
		if err != nil {
			log.Fatal(err)
			return
		}

		fmt.Println("synced successfully")
	},
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Run this to add new vercel client account",
	Run: func(cmd *cobra.Command, args []string) {
		err := NewAccountCmd()
		if err != nil {
			log.Fatal(err)
			return
		}

		fmt.Println("you can now add new account using `vercel login` and then run `vercelgate sync` again")
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset vercelgate state and will delete all accounts",
	Run: func(cmd *cobra.Command, args []string) {
		err := ResetCmd()
		if err != nil {
			log.Fatal(err)
			return
		}

		fmt.Println("state reset was successful")
	},
}

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show Vercel global configuration path",
	Run: func(cmd *cobra.Command, args []string) {
		globalPath, err := vercelutil.GetGlobalPathConfig()
		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Printf("Vercel global configuration path: %s\n", globalPath)
	},
}

func NewAccountCmd() error {
	filePath, err := vercelutil.AuthJsonFile()
	if err != nil {
		return err
	}

	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove auth.json file: %w", err)
	}
	return nil
}

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List synced accounts and their teams as JSON (no tokens)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ListAccounts(); err != nil {
			log.Fatal(err)
			return
		}
	},
}

func ListAccounts() error {
	ctx := context.Background()

	users, err := entdb.Client().User.Query().All(ctx)
	if err != nil {
		return err
	}

	type teamOut struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	type userOut struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
		Teams    []teamOut `json:"teams"`
	}

	out := []userOut{}
	for _, u := range users {
		teams, err := entdb.Client().Team.Query().Where(team.UserID(u.ID)).All(ctx)
		if err != nil {
			return err
		}
		to := []teamOut{}
		for _, t := range teams {
			to = append(to, teamOut{ID: t.ID, Name: t.Name, Slug: t.Slug})
		}
		out = append(out, userOut{ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, Teams: to})
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func ResetCmd() error {
	ctx := context.Background()
	_, err := entdb.Client().User.Delete().Exec(ctx)
	if err != nil {
		return err
	}
	_, err = entdb.Client().Team.Delete().Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}
