package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/config"
	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up ComplexityRadar for your project",
	Long: `Interactive wizard to configure ComplexityRadar.

Creates a project and repository configuration in the local database.

Running 'radar init' again will create a new project. Use --db to specify
a different database file if you need multiple configurations.

Example:
  radar init`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().String("db", ".complexity-radar.db", "Database file path")
}

func runInit(cmd *cobra.Command, args []string) error {
	dbPath, _ := cmd.Flags().GetString("db")

	s, err := openStore(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	reader := bufio.NewReader(os.Stdin)

	project, err := promptProject(reader, s)
	if err != nil {
		return err
	}

	_, err = promptRepositories(reader, s, project.ID)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nConfiguration saved to %s\n", dbPath)
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'radar scan' to analyze your project.")

	return nil
}

func promptProject(reader *bufio.Reader, s *store.Store) (*model.Project, error) {
	fmt.Println("Let's set up ComplexityRadar for your project.")

	existing, err := s.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	if len(existing) > 0 {
		fmt.Println("\nExisting projects:")
		for i, p := range existing {
			fmt.Printf("  %d. %s\n", i+1, p.Name)
		}

		choice, err := prompt(reader, "\nSelect project number or press Enter to create new", "")
		if err != nil {
			return nil, err
		}

		if choice != "" {
			var idx int
			if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(existing) {
				selected := &existing[idx-1]
				key, clientID, clientSecret, err := promptDevCycle(reader, selected.DevCycleProjectKey, selected.DevCycleClientID, selected.DevCycleClientSecret)
				if err != nil {
					return nil, err
				}
				if key != selected.DevCycleProjectKey || clientID != selected.DevCycleClientID || clientSecret != selected.DevCycleClientSecret {
					selected.DevCycleProjectKey = key
					selected.DevCycleClientID = clientID
					selected.DevCycleClientSecret = clientSecret
					if err := s.UpdateProject(selected); err != nil {
						return nil, fmt.Errorf("updating project: %w", err)
					}
				}
				return selected, nil
			}
			fmt.Println("Invalid selection, creating new project.")
		}
	}

	fmt.Println()
	name, err := prompt(reader, "Project name", "")
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}

	description, err := prompt(reader, "Description (optional)", "")
	if err != nil {
		return nil, err
	}

	devCycleKey, devCycleClientID, devCycleClientSecret, err := promptDevCycle(reader, "", "", "")
	if err != nil {
		return nil, err
	}

	project := &model.Project{
		Name:                 name,
		Description:          description,
		DevCycleProjectKey:   devCycleKey,
		DevCycleClientID:     devCycleClientID,
		DevCycleClientSecret: devCycleClientSecret,
	}
	if err := s.CreateProject(project); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	fmt.Printf("\nProject '%s' created.\n", name)
	return project, nil
}

func promptRepositories(reader *bufio.Reader, s *store.Store, projectID int64) ([]model.Repository, error) {
	existing, err := s.ListRepositories(projectID)
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	if len(existing) > 0 {
		for {
			fmt.Println("\nRepositories:")
			for i, r := range existing {
				fmt.Printf("  %d. %s (branch: %s)\n", i+1, r.URL, r.Branch)
			}

			choice, err := prompt(reader, "\nEnter to continue, number to edit, [a]dd new", "")
			if err != nil {
				return nil, err
			}

			if choice == "" {
				break
			}

			if strings.ToLower(choice) == "a" {
				repo, err := addRepository(reader, s, projectID)
				if err != nil {
					return nil, err
				}
				if repo != nil {
					existing = append(existing, *repo)
				}
				continue
			}

			var idx int
			if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(existing) {
				updated, err := editRepository(reader, s, &existing[idx-1])
				if err != nil {
					return nil, err
				}
				existing[idx-1] = *updated
				continue
			}

			fmt.Println("Invalid selection.")
		}
	} else {
		for {
			repo, err := addRepository(reader, s, projectID)
			if err != nil {
				return nil, err
			}
			if repo == nil {
				break
			}
			existing = append(existing, *repo)
		}
	}

	return existing, nil
}

func addRepository(reader *bufio.Reader, s *store.Store, projectID int64) (*model.Repository, error) {
	fmt.Println()
	url, err := prompt(reader, "Repository URL (e.g. github.com/org/repo)", "")
	if err != nil {
		return nil, err
	}
	if url == "" {
		return nil, nil
	}

	if !config.IsValidRepoURL(url) {
		fmt.Println("Invalid URL. Expected format: github.com/org/repo")
		return addRepository(reader, s, projectID)
	}

	branch, err := prompt(reader, "Branch", "main")
	if err != nil {
		return nil, err
	}

	method, err := promptDeployMethod(reader, config.DeployDetectionReleases)
	if err != nil {
		return nil, err
	}

	var includePrereleases bool
	if method == config.DeployDetectionReleases {
		includePrereleases, err = promptYesNo(reader, "Count prereleases (betas/RCs) as deploys?", false)
		if err != nil {
			return nil, err
		}
	}

	tagPrefix, err := prompt(reader, tagPrefixLabel(method), "")
	if err != nil {
		return nil, err
	}

	repo := &model.Repository{
		ProjectID:          projectID,
		URL:                url,
		Branch:             branch,
		DeployDetection:    method,
		IncludePrereleases: includePrereleases,
		TagPrefix:          tagPrefix,
	}
	if err := s.CreateRepository(repo); err != nil {
		return nil, fmt.Errorf("creating repository: %w", err)
	}

	fmt.Printf("Repository '%s' added.\n", url)
	return repo, nil
}

func editRepository(reader *bufio.Reader, s *store.Store, repo *model.Repository) (*model.Repository, error) {
	fmt.Printf("\nEditing %s\n", repo.URL)

	url, err := prompt(reader, "Repository URL", repo.URL)
	if err != nil {
		return nil, err
	}

	branch, err := prompt(reader, "Branch", repo.Branch)
	if err != nil {
		return nil, err
	}

	method, err := promptDeployMethod(reader, repo.DeployDetection)
	if err != nil {
		return nil, err
	}

	var includePrereleases bool
	if method == config.DeployDetectionReleases {
		includePrereleases, err = promptYesNo(reader, "Count prereleases (betas/RCs) as deploys?", repo.IncludePrereleases)
		if err != nil {
			return nil, err
		}
	}

	tagPrefix, err := prompt(reader, tagPrefixLabel(method), repo.TagPrefix)
	if err != nil {
		return nil, err
	}

	repo.URL = url
	repo.Branch = branch
	repo.DeployDetection = method
	repo.IncludePrereleases = includePrereleases
	repo.TagPrefix = tagPrefix

	if err := s.UpdateRepository(repo); err != nil {
		return nil, fmt.Errorf("updating repository: %w", err)
	}

	fmt.Printf("Repository updated.\n")
	return repo, nil
}

func promptDevCycle(reader *bufio.Reader, key, clientID, clientSecret string) (nKey, nClientID, nClientSecret string, err error) {
	track, err := promptYesNo(reader, "Track DevCycle feature-flag debt for this project?", key != "")
	if err != nil {
		return "", "", "", err
	}
	if !track {
		return "", "", "", nil
	}

	key, err = prompt(reader, "DevCycle project key (from the /p/<key> URL segment)", key)
	if err != nil {
		return "", "", "", err
	}
	clientID, err = prompt(reader, "DevCycle client ID (app.devcycle.com/r/settings)", clientID)
	if err != nil {
		return "", "", "", err
	}
	clientSecret, err = prompt(reader, "DevCycle client secret", clientSecret)
	if err != nil {
		return "", "", "", err
	}
	return key, clientID, clientSecret, nil
}

func promptDeployMethod(reader *bufio.Reader, current string) (string, error) {
	fmt.Println("How are deployments detected?")
	fmt.Println("  1. GitHub Releases")
	fmt.Println("  2. Git tags")

	def := "1"
	if current == config.DeployDetectionTags {
		def = "2"
	}
	choice, err := prompt(reader, "Select", def)
	if err != nil {
		return "", err
	}
	if choice == "2" {
		return config.DeployDetectionTags, nil
	}
	return config.DeployDetectionReleases, nil
}

func tagPrefixLabel(method string) string {
	if method == config.DeployDetectionTags {
		return "Only count tags matching prefix (optional, e.g. promote/)"
	}
	return "Only count releases with tag prefix (optional, e.g. v)"
}

func promptYesNo(reader *bufio.Reader, label string, def bool) (bool, error) {
	d := "n"
	if def {
		d = "y"
	}
	input, err := prompt(reader, label+" (y/n)", d)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(input) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func prompt(reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}

	return input, nil
}
