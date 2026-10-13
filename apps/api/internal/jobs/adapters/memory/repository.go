package memory

import (
	"context"
	"sync"

	"github.com/nickdavid1811/imageflow/internal/jobs/application"
	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

type JobRepository struct {
	mu   sync.RWMutex
	jobs map[string]domain.Job
}

func NewJobRepository() *JobRepository {
	return &JobRepository{jobs: make(map[string]domain.Job)}
}

func (repository *JobRepository) Create(_ context.Context, job domain.Job) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.jobs[job.ID] = job
	return nil
}

func (repository *JobRepository) FindByID(_ context.Context, id string) (domain.Job, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	job, ok := repository.jobs[id]
	if !ok {
		return domain.Job{}, application.ErrJobNotFound
	}
	return job, nil
}

func (repository *JobRepository) FindByProjectID(ctx context.Context, projectID, id string) (domain.Job, error) {
	job, err := repository.FindByID(ctx, id)
	if err != nil {
		return domain.Job{}, err
	}
	if job.ProjectID != projectID {
		return domain.Job{}, application.ErrJobNotFound
	}
	return job, nil
}
