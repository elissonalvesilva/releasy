package store

import (
	"context"
	"embed"
	"github.com/elissonalvesilva/releasy/internal/core/dto"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrations embed.FS

type PgStore struct {
	DB *sqlx.DB
}

type DbStore interface {
	SaveDeployment(ctx context.Context, d dto.Deployment) error
	GetDeployments(ctx context.Context, serviceName string, limit int) ([]Deployment, error)
	DeleteOldDeployments(ctx context.Context, serviceName string, keepLast int) error
	UpdateDeploymentStep(ctx context.Context, id string, step string) error
	GetDeploymentByID(ctx context.Context, id string) (*Deployment, error)

	SaveSlot(ctx context.Context, s Slot) error
	GetSlots(ctx context.Context, serviceName string) ([]Slot, error)
	GetSlot(ctx context.Context, serviceName, version string) (*Slot, error)
	DeleteSlot(ctx context.Context, serviceName, version string) error

	SaveEvent(ctx context.Context, e Event) error
	GetEvents(ctx context.Context, serviceName string, limit int) ([]Event, error)
}

type Deployment struct {
	ID          string    `db:"id"`
	ServiceName string    `db:"service_name"`
	Strategy    string    `db:"strategy"`
	Version     string    `db:"version"`
	Replicas    int       `db:"replicas"`
	Image       string    `db:"image"`
	Action      string    `db:"action"`
	Step        string    `db:"step"`
	Envs        string    `db:"envs"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Slot struct {
	ID          string    `db:"id"`
	ServiceName string    `db:"service_name"`
	Version     string    `db:"version"`
	Weight      int       `db:"weight"`
	CreatedAt   time.Time `db:"created_at"`
}

type Event struct {
	ID          string    `db:"id"`
	ServiceName string    `db:"service_name"`
	Message     string    `db:"message"`
	CreatedAt   time.Time `db:"created_at"`
}

func NewPgStore(dsn string) (*PgStore, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PgStore{DB: db}, nil
}

func (s *PgStore) InitSchema(ctx context.Context) error {
	schemaSQL, err := migrations.ReadFile("migrations/init.sql")
	if err != nil {
		return err
	}

	_, err = s.DB.ExecContext(ctx, string(schemaSQL))
	return err
}

func (s *PgStore) SaveDeployment(ctx context.Context, d dto.Deployment) error {
	query := `
		INSERT INTO deployments (
			id, service_name, strategy, version,
			replicas, image, action, step, envs, created_at
		) VALUES (
			:id, :service_name, :strategy, :version,
			:replicas, :image, :action, :step, :envs, :created_at
		)
	`
	model := s.toModel(d)

	_, err := s.DB.NamedExecContext(ctx, query, model)
	return err
}

func (s *PgStore) GetDeployments(ctx context.Context, serviceName string, limit int) ([]Deployment, error) {
	var deploys []Deployment
	query := `
		SELECT * FROM deployments
		WHERE service_name = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	err := s.DB.SelectContext(ctx, &deploys, query, serviceName, limit)
	return deploys, err
}

func (s *PgStore) DeleteOldDeployments(ctx context.Context, serviceName string, keepLast int) error {
	query := `
		DELETE FROM deployments
		WHERE id NOT IN (
			SELECT id FROM deployments
			WHERE service_name = $1
			ORDER BY created_at DESC
			LIMIT $2
		) AND service_name = $1
	`
	_, err := s.DB.ExecContext(ctx, query, serviceName, keepLast)
	return err
}

func (s *PgStore) UpdateDeploymentStep(ctx context.Context, id string, step string) error {
	query := `UPDATE deployments SET step = $1 WHERE id = $2`
	_, err := s.DB.ExecContext(ctx, query, step, id)
	return err
}

func (s *PgStore) GetDeploymentByID(ctx context.Context, id string) (*Deployment, error) {
	var d Deployment
	query := `SELECT * FROM deployments WHERE id = $1`
	err := s.DB.GetContext(ctx, &d, query, id)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *PgStore) SaveSlot(ctx context.Context, slot Slot) error {
	query := `
		INSERT INTO slots (id, service_name, version, weight, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (service_name, version) DO UPDATE
		SET weight = EXCLUDED.weight, created_at = EXCLUDED.created_at
	`
	_, err := s.DB.ExecContext(ctx, query,
		slot.ID, slot.ServiceName, slot.Version, slot.Weight, slot.CreatedAt)
	return err
}

func (s *PgStore) GetSlots(ctx context.Context, serviceName string) ([]Slot, error) {
	var slots []Slot
	query := `SELECT * FROM slots WHERE service_name = $1 ORDER BY created_at DESC`
	err := s.DB.SelectContext(ctx, &slots, query, serviceName)
	return slots, err
}

func (s *PgStore) GetSlot(ctx context.Context, serviceName, version string) (*Slot, error) {
	var slot Slot
	query := `SELECT * FROM slots WHERE service_name = $1 AND version = $2`
	err := s.DB.GetContext(ctx, &slot, query, serviceName, version)
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (s *PgStore) DeleteSlot(ctx context.Context, serviceName, version string) error {
	query := `DELETE FROM slots WHERE service_name = $1 AND version = $2`
	_, err := s.DB.ExecContext(ctx, query, serviceName, version)
	return err
}

func (s *PgStore) SaveEvent(ctx context.Context, e Event) error {
	query := `INSERT INTO events (id, service_name, message, created_at) VALUES ($1, $2, $3, $4)`
	_, err := s.DB.ExecContext(ctx, query, e.ID, e.ServiceName, e.Message, e.CreatedAt)
	return err
}

func (s *PgStore) GetEvents(ctx context.Context, serviceName string, limit int) ([]Event, error) {
	var events []Event
	query := `SELECT * FROM events WHERE service_name = $1 ORDER BY created_at DESC LIMIT $2`
	err := s.DB.SelectContext(ctx, &events, query, serviceName, limit)
	return events, err
}

func (s *PgStore) toModel(d dto.Deployment) Deployment {
	return Deployment{
		ID:          d.ID,
		ServiceName: d.ServiceName,
		Strategy:    d.DeploymentStrategy,
		Version:     d.Version,
		Replicas:    d.Replicas,
		Image:       d.Image,
		Action:      d.Action,
		Step:        d.Step,
		Envs:        d.Envs,
		CreatedAt:   d.CreatedAt,
	}
}
