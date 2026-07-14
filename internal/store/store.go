package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kobbikobb/complexity-radar/internal/model"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store provides SQLite storage for ComplexityRadar.
type Store struct {
	db *sql.DB
}

// New opens a database at dbPath, runs migrations, and returns a Store.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	if err := s.EnsureMetricTypes(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seeding metric types: %w", err)
	}

	return s, nil
}

// NewFromDB creates a Store from an existing *sql.DB (for testing).
func NewFromDB(db *sql.DB) (*Store, error) {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	if err := s.EnsureMetricTypes(); err != nil {
		return nil, fmt.Errorf("seeding metric types: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)"); err != nil {
		return fmt.Errorf("creating migration tracking table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		var applied int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", name).Scan(&applied); err != nil {
			return fmt.Errorf("checking migration %s: %w", name, err)
		}
		if applied > 0 {
			continue
		}

		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}

		if _, err := s.db.Exec(string(data)); err != nil {
			return fmt.Errorf("executing migration %s: %w", name, err)
		}

		if _, err := s.db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name); err != nil {
			return fmt.Errorf("recording migration %s: %w", name, err)
		}
	}

	return nil
}

func (s *Store) CreateProject(p *model.Project) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO projects (name, description, created_at, updated_at) VALUES (?, ?, ?, ?)",
		p.Name, p.Description, now, now,
	)
	if err != nil {
		return fmt.Errorf("inserting project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting project id: %w", err)
	}

	p.ID = id
	p.CreatedAt, _ = time.Parse(time.RFC3339, now)
	p.UpdatedAt = p.CreatedAt
	return nil
}

func (s *Store) GetProjectByName(name string) (*model.Project, error) {
	p := &model.Project{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		"SELECT id, name, description, created_at, updated_at FROM projects WHERE name = ?", name,
	).Scan(&p.ID, &p.Name, &p.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project %q: %w", name, ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("querying project by name: %w", err)
	}

	p.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	p.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	return p, nil
}

func (s *Store) GetProject(id int64) (*model.Project, error) {
	p := &model.Project{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		"SELECT id, name, description, created_at, updated_at FROM projects WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Description, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("querying project: %w", err)
	}

	p.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	p.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	return p, nil
}

func (s *Store) ListProjects() ([]model.Project, error) {
	rows, err := s.db.Query("SELECT id, name, description, created_at, updated_at FROM projects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("querying projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var createdAt, updatedAt string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning project: %w", err)
		}
		p.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parsing created_at: %w", err)
		}
		p.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing updated_at: %w", err)
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}

func (s *Store) UpdateProject(p *model.Project) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"UPDATE projects SET name = ?, description = ?, updated_at = ? WHERE id = ?",
		p.Name, p.Description, now, p.ID,
	)
	if err != nil {
		return fmt.Errorf("updating project: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project %d not found", p.ID)
	}

	p.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (s *Store) DeleteProject(id int64) error {
	result, err := s.db.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project %d not found", id)
	}

	return nil
}

func (s *Store) CreateRepository(r *model.Repository) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO repositories (project_id, url, branch, gitops_repo_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		r.ProjectID, r.URL, r.Branch, r.GitopsRepoURL, now, now,
	)
	if err != nil {
		return fmt.Errorf("inserting repository: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting repository id: %w", err)
	}

	r.ID = id
	r.CreatedAt, _ = time.Parse(time.RFC3339, now)
	r.UpdatedAt = r.CreatedAt
	return nil
}

// FindOrCreateRepository returns an existing repository matching projectID and URL,
// or creates a new one if none exists. If the branch differs, it is updated.
func (s *Store) FindOrCreateRepository(projectID int64, url, branch string) (*model.Repository, error) {
	r := &model.Repository{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		"SELECT id, project_id, url, branch, created_at, updated_at FROM repositories WHERE project_id = ? AND url = ?",
		projectID, url,
	).Scan(&r.ID, &r.ProjectID, &r.URL, &r.Branch, &createdAt, &updatedAt)
	if err == nil {
		r.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parsing created_at: %w", err)
		}
		r.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing updated_at: %w", err)
		}

		if r.Branch != branch {
			now := time.Now().UTC().Format(time.RFC3339)
			if _, err := s.db.Exec(
				"UPDATE repositories SET branch = ?, updated_at = ? WHERE id = ?",
				branch, now, r.ID,
			); err != nil {
				return nil, fmt.Errorf("updating repository branch: %w", err)
			}
			r.Branch = branch
			r.UpdatedAt, _ = time.Parse(time.RFC3339, now)
		}

		return r, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("querying repository: %w", err)
	}

	repo := &model.Repository{
		ProjectID: projectID,
		URL:       url,
		Branch:    branch,
	}
	if err := s.CreateRepository(repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func (s *Store) GetRepository(id int64) (*model.Repository, error) {
	r := &model.Repository{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		"SELECT id, project_id, url, branch, gitops_repo_url, created_at, updated_at FROM repositories WHERE id = ?", id,
	).Scan(&r.ID, &r.ProjectID, &r.URL, &r.Branch, &r.GitopsRepoURL, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("repository %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("querying repository: %w", err)
	}

	r.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at: %w", err)
	}
	r.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at: %w", err)
	}
	return r, nil
}

func (s *Store) ListRepositories(projectID int64) ([]model.Repository, error) {
	rows, err := s.db.Query(
		"SELECT id, project_id, url, branch, gitops_repo_url, created_at, updated_at FROM repositories WHERE project_id = ? ORDER BY id",
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying repositories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var repos []model.Repository
	for rows.Next() {
		var r model.Repository
		var createdAt, updatedAt string
		if err := rows.Scan(&r.ID, &r.ProjectID, &r.URL, &r.Branch, &r.GitopsRepoURL, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning repository: %w", err)
		}
		r.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parsing created_at: %w", err)
		}
		r.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing updated_at: %w", err)
		}
		repos = append(repos, r)
	}

	return repos, rows.Err()
}

func (s *Store) UpdateRepository(r *model.Repository) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"UPDATE repositories SET url = ?, branch = ?, gitops_repo_url = ?, updated_at = ? WHERE id = ?",
		r.URL, r.Branch, r.GitopsRepoURL, now, r.ID,
	)
	if err != nil {
		return fmt.Errorf("updating repository: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("repository %d not found", r.ID)
	}

	r.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (s *Store) DeleteRepository(id int64) error {
	result, err := s.db.Exec("DELETE FROM repositories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("repository %d not found", id)
	}

	return nil
}

func (s *Store) EnsureMetricTypes() error {
	for _, mt := range model.MetricTypes() {
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO metric_types (name, dimension, unit) VALUES (?, ?, ?)",
			string(mt.Name), string(mt.Dimension), mt.Unit,
		)
		if err != nil {
			return fmt.Errorf("ensuring metric type %s: %w", mt.Name, err)
		}
	}
	for _, mt := range model.DisplayMetricTypes() {
		_, err := s.db.Exec(
			"INSERT OR IGNORE INTO metric_types (name, dimension, unit) VALUES (?, ?, ?)",
			string(mt.Name), string(mt.Dimension), mt.Unit,
		)
		if err != nil {
			return fmt.Errorf("ensuring display metric type %s: %w", mt.Name, err)
		}
	}
	return nil
}

func (s *Store) GetMetricTypeByName(name model.MetricTypeName) (*model.MetricType, error) {
	mt := &model.MetricType{}
	err := s.db.QueryRow(
		"SELECT id, name, dimension, unit FROM metric_types WHERE name = ?", string(name),
	).Scan(&mt.ID, &mt.Name, &mt.Dimension, &mt.Unit)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("metric type %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("querying metric type: %w", err)
	}

	return mt, nil
}

func (s *Store) GetMetricTypeByID(id int64) (*model.MetricType, error) {
	mt := &model.MetricType{}
	err := s.db.QueryRow(
		"SELECT id, name, dimension, unit FROM metric_types WHERE id = ?", id,
	).Scan(&mt.ID, &mt.Name, &mt.Dimension, &mt.Unit)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("metric type %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("querying metric type: %w", err)
	}

	return mt, nil
}

func (s *Store) CreateMetric(m *model.Metric) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO metrics (repository_id, metric_type_id, value, collected_at) VALUES (?, ?, ?, ?)",
		m.RepositoryID, m.MetricTypeID, m.Value, now,
	)
	if err != nil {
		return fmt.Errorf("inserting metric: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting metric id: %w", err)
	}

	m.ID = id
	m.CollectedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (s *Store) GetMetricsByRepository(repoID int64) ([]model.Metric, error) {
	rows, err := s.db.Query(
		"SELECT id, repository_id, metric_type_id, value, collected_at FROM metrics WHERE repository_id = ? ORDER BY id",
		repoID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying metrics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var metrics []model.Metric
	for rows.Next() {
		var m model.Metric
		var collectedAt string
		if err := rows.Scan(&m.ID, &m.RepositoryID, &m.MetricTypeID, &m.Value, &collectedAt); err != nil {
			return nil, fmt.Errorf("scanning metric: %w", err)
		}
		m.CollectedAt, err = time.Parse(time.RFC3339, collectedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing collected_at: %w", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

func (s *Store) CreateDimensionScore(ds *model.DimensionScore) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO dimension_scores (repository_id, dimension, score, weight, computed_at) VALUES (?, ?, ?, ?, ?)",
		ds.RepositoryID, string(ds.Dimension), ds.Score, ds.Weight, now,
	)
	if err != nil {
		return fmt.Errorf("inserting dimension score: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting dimension score id: %w", err)
	}

	ds.ID = id
	ds.ComputedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (s *Store) GetDimensionScoresByRepository(repoID int64) ([]model.DimensionScore, error) {
	rows, err := s.db.Query(
		"SELECT id, repository_id, dimension, score, weight, computed_at FROM dimension_scores WHERE repository_id = ? ORDER BY id",
		repoID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying dimension scores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var scores []model.DimensionScore
	for rows.Next() {
		var ds model.DimensionScore
		var dimension, computedAt string
		if err := rows.Scan(&ds.ID, &ds.RepositoryID, &dimension, &ds.Score, &ds.Weight, &computedAt); err != nil {
			return nil, fmt.Errorf("scanning dimension score: %w", err)
		}
		ds.Dimension = model.Dimension(dimension)
		ds.ComputedAt, err = time.Parse(time.RFC3339, computedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing computed_at: %w", err)
		}
		scores = append(scores, ds)
	}

	return scores, rows.Err()
}

func (s *Store) CreateProjectReport(pr *model.ProjectReport) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		"INSERT INTO project_reports (project_id, total_score, computed_at) VALUES (?, ?, ?)",
		pr.ProjectID, pr.TotalScore, now,
	)
	if err != nil {
		return fmt.Errorf("inserting project report: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting project report id: %w", err)
	}

	pr.ID = id
	pr.ComputedAt, _ = time.Parse(time.RFC3339, now)
	return nil
}

func (s *Store) GetProjectReport(id int64) (*model.ProjectReport, error) {
	pr := &model.ProjectReport{}
	var computedAt string
	err := s.db.QueryRow(
		"SELECT id, project_id, total_score, computed_at FROM project_reports WHERE id = ?", id,
	).Scan(&pr.ID, &pr.ProjectID, &pr.TotalScore, &computedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project report %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("querying project report: %w", err)
	}

	pr.ComputedAt, err = time.Parse(time.RFC3339, computedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing computed_at: %w", err)
	}
	return pr, nil
}

func (s *Store) AddProjectReportScore(prs *model.ProjectReportScore) error {
	result, err := s.db.Exec(
		"INSERT INTO project_report_scores (project_report_id, dimension, score, weight) VALUES (?, ?, ?, ?)",
		prs.ProjectReportID, string(prs.Dimension), prs.Score, prs.Weight,
	)
	if err != nil {
		return fmt.Errorf("inserting project report score: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting project report score id: %w", err)
	}

	prs.ID = id
	return nil
}

func (s *Store) GetProjectReportScores(reportID int64) ([]model.ProjectReportScore, error) {
	rows, err := s.db.Query(
		"SELECT id, project_report_id, dimension, score, weight FROM project_report_scores WHERE project_report_id = ? ORDER BY id",
		reportID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying project report scores: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var scores []model.ProjectReportScore
	for rows.Next() {
		var prs model.ProjectReportScore
		var dimension string
		if err := rows.Scan(&prs.ID, &prs.ProjectReportID, &dimension, &prs.Score, &prs.Weight); err != nil {
			return nil, fmt.Errorf("scanning project report score: %w", err)
		}
		prs.Dimension = model.Dimension(dimension)
		scores = append(scores, prs)
	}

	return scores, rows.Err()
}

// MigrateSchema is exported for tests that need to verify migration.
func (s *Store) MigrateSchema() error {
	return s.migrate()
}
