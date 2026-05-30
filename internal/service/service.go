package service

import (
	"fmt"
	"os"
	"time"

	"kodo/internal/analytics"
	"kodo/internal/auth"
	"kodo/internal/config"
	"kodo/internal/database"
	"kodo/internal/models"
	"kodo/internal/scraper"
)

type Service struct {
	cfg    *config.Config
	auth   auth.Authenticator
	scraper *scraper.Scraper
	db     *database.DB
	engine *analytics.Engine
}

type ServiceData struct {
	Usage   *models.WorkspaceUsage
	Models  *models.ModelSummary
	Context *models.ContextStats
	Summary *Summary
}

type Summary struct {
	HealthStatus   string
	LastSync      time.Time
	AuthMode      string
	Workspace     string
	Error         error
}

func New(cfg *config.Config) (*Service, error) {
	svc := &Service{
		cfg:  cfg,
		auth: auth.NewAuthenticator(cfg),
	}

	engine := analytics.New()
	svc.engine = engine

	return svc, nil
}

func (s *Service) Authenticate() error {
	cookie, err := s.auth.GetCookie()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if s.cfg.Session.Workspace == "" {
		return fmt.Errorf("workspace not configured")
	}

	s.scraper = scraper.NewScraper(s.cfg.Session.Workspace, cookie)

	return nil
}

func (s *Service) AuthenticateManual() error {
	cookie, err := s.promptForCookie()
	if err != nil {
		return err
	}

	s.cfg.Session.Cookie = cookie
	s.cfg.Auth.Mode = "manual"

	if err := s.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	s.scraper = scraper.NewScraper(s.cfg.Session.Workspace, cookie)

	return nil
}

func (s *Service) promptForCookie() (string, error) {
	fmt.Print("Enter your OpenCode cookie (__session token): ")
	var cookie string
	fmt.Scanln(&cookie)

	if cookie == "" {
		cookie = os.Getenv("OPENCODE_COOKIE")
	}

	if cookie == "" {
		return "", fmt.Errorf("no cookie provided")
	}

	return cookie, nil
}

func (s *Service) ConnectDB() error {
	var err error
	s.db, err = database.New(s.cfg.Paths.LocalDB)
	if err != nil {
		return fmt.Errorf("failed to connect to local database: %w", err)
	}
	return nil
}

func (s *Service) LoadAllData() (*ServiceData, error) {
	data := &ServiceData{
		Summary: &Summary{
			AuthMode:  s.cfg.Auth.Mode,
			Workspace: s.cfg.Session.Workspace,
			LastSync:  time.Now(),
		},
	}

	if s.scraper != nil {
		usage, err := s.scraper.FetchUsage()
		if err != nil {
			data.Summary.Error = fmt.Errorf("cloud fetch failed: %w", err)
		} else {
			data.Usage = usage
			s.engine.WithUsage(usage)
		}
	}

	if s.db != nil {
		summary, err := s.db.FetchModelSummary()
		if err == nil && summary != nil {
			data.Models = summary
			s.engine.WithModels(summary)
		}

		ctxStats, err := s.db.FetchContextStats()
		if err == nil && ctxStats != nil {
			data.Context = ctxStats
			s.engine.WithContext(ctxStats)
		}
	}

	if data.Usage != nil {
		s.engine.CalculateBurnRate()
		data.Summary.HealthStatus = s.engine.GetHealthSummary()
	}

	return data, nil
}

func (s *Service) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Service) Config() *config.Config {
	return s.cfg
}
