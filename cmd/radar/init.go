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
				return &existing[idx-1], nil
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

	project := &model.Project{
		Name:        name,
		Description: description,
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

	detection, err := promptDeployDetection(reader)
	if err != nil {
		return nil, err
	}

	repo := &model.Repository{
		ProjectID:       projectID,
		URL:             url,
		Branch:          branch,
		DeployDetection: detection,
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

	detection, err := promptDeployDetection(reader)
	if err != nil {
		return nil, err
	}

	repo.URL = url
	repo.Branch = branch
	repo.DeployDetection = detection

	if err := s.UpdateRepository(repo); err != nil {
		return nil, fmt.Errorf("updating repository: %w", err)
	}

	fmt.Printf("Repository updated.\n")
	return repo, nil
}

func promptDeployDetection(reader *bufio.Reader) (string, error) {
	fmt.Println("Deploy detection method:")
	fmt.Println("  1. GitHub Releases (default)")
	fmt.Println("  [2. git tags — coming soon]")

	if _, err := prompt(reader, "Select", "1"); err != nil {
		return "", err
	}
	return config.DeployDetectionReleases, nil
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
