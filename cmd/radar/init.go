package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kobbikobb/complexity-radar/internal/model"
	"github.com/kobbikobb/complexity-radar/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up ComplexityRadar for your project",
	Long: `Interactive wizard to configure ComplexityRadar.

Creates a project and repository configuration in the local database.

Example:
  radar init`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	s, err := store.New(".complexity-radar.db")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
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

	fmt.Printf("\nConfiguration saved to .complexity-radar.db\n")
	fmt.Printf("Run 'radar scan' to analyze your project.\n")

	return nil
}

func promptProject(reader *bufio.Reader, s *store.Store) (*model.Project, error) {
	fmt.Println("Let's set up ComplexityRadar for your project.")

	existing, err := s.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	if len(existing) > 0 {
		fmt.Println("Existing projects:")
		for i, p := range existing {
			fmt.Printf("  %d. %s\n", i+1, p.Name)
		}
		fmt.Println()
	}

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
	fmt.Println("\nNow let's add your repositories.")

	var repos []model.Repository
	for {
		url, err := prompt(reader, "Repository URL (e.g. github.com/org/repo)", "")
		if err != nil {
			return nil, err
		}
		if url == "" {
			if len(repos) == 0 {
				fmt.Println("At least one repository is required.")
				continue
			}
			break
		}

		branch, err := prompt(reader, "Branch", "main")
		if err != nil {
			return nil, err
		}

		repo := &model.Repository{
			ProjectID: projectID,
			URL:       url,
			Branch:    branch,
		}
		if err := s.CreateRepository(repo); err != nil {
			return nil, fmt.Errorf("creating repository: %w", err)
		}

		repos = append(repos, *repo)
		fmt.Printf("Repository '%s' added.\n\n", url)
	}

	return repos, nil
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
