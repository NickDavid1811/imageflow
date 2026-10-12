package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

var ErrJobNotFound = errors.New("job not found")

// JobRepository is the persistence port required by JobService.
type JobRepository interface {
	Create(context.Context, domain.Job) error
	FindByID(context.Context, string) (domain.Job, error)
}

// CreateJobCommand contains input for a new job.
type CreateJobCommand struct {
	SourceURL  string
	Operations []domain.Operation
}

// JobService coordinates job creation and retrieval.
type JobService struct {
	repository JobRepository
	now        func() time.Time
	newID      func() (string, error)
}

// NewJobService constructs a service with production clock and ID generation.
func NewJobService(repository JobRepository) *JobService {
	return &JobService{
		repository: repository,
		now:        time.Now,
		newID:      randomID,
	}
}

// Create validates and persists a new queued job.
func (service *JobService) Create(ctx context.Context, command CreateJobCommand) (domain.Job, error) {
	id, err := service.newID()
	if err != nil {
		return domain.Job{}, fmt.Errorf("generate job id: %w", err)
	}
	job, err := domain.NewJob(id, command.SourceURL, command.Operations, service.now())
	if err != nil {
		return domain.Job{}, err
	}
	if err := service.repository.Create(ctx, job); err != nil {
		return domain.Job{}, fmt.Errorf("store job: %w", err)
	}
	return job, nil
}

// Get returns a job by identifier or ErrJobNotFound.
func (service *JobService) Get(ctx context.Context, id string) (domain.Job, error) {
	return service.repository.FindByID(ctx, id)
}

func randomID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "job_" + hex.EncodeToString(bytes), nil
}
