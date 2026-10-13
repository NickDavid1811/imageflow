package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/nickdavid1811/imageflow/internal/jobs/domain"
)

var ErrJobNotFound = errors.New("job not found")
var ErrResultNotReady = errors.New("job result is not ready")

// JobRepository is the persistence port required by JobService.
type JobRepository interface {
	Create(context.Context, domain.Job) error
	FindByProjectID(context.Context, string, string) (domain.Job, error)
}

// CreateJobCommand contains input for a new job.
type CreateJobCommand struct {
	ProjectID  string
	SourceURL  string
	Operations []domain.Operation
}

// JobService coordinates job creation and retrieval.
type JobService struct {
	repository JobRepository
	outputs    OutputReader
	now        func() time.Time
	newID      func() (string, error)
}

// OutputReader opens generated artifacts by storage key.
type OutputReader interface {
	Open(context.Context, string) (io.ReadCloser, error)
}

// JobResult contains a stream and its public metadata.
type JobResult struct {
	Body   io.ReadCloser
	Output domain.Output
}

// NewJobService constructs a service with production clock and ID generation.
func NewJobService(repository JobRepository, outputs ...OutputReader) *JobService {
	var outputReader OutputReader
	if len(outputs) > 0 {
		outputReader = outputs[0]
	}
	return &JobService{
		repository: repository,
		outputs:    outputReader,
		now:        time.Now,
		newID:      randomID,
	}
}

// Result opens the generated artifact of a completed job.
func (service *JobService) Result(ctx context.Context, projectID, id string) (JobResult, error) {
	job, err := service.repository.FindByProjectID(ctx, projectID, id)
	if err != nil {
		return JobResult{}, err
	}
	if job.Status != domain.StatusCompleted || job.Output == nil || service.outputs == nil {
		return JobResult{}, ErrResultNotReady
	}
	body, err := service.outputs.Open(ctx, job.Output.Key)
	if err != nil {
		return JobResult{}, fmt.Errorf("open job result: %w", err)
	}
	return JobResult{Body: body, Output: *job.Output}, nil
}

// Create validates and persists a new queued job.
func (service *JobService) Create(ctx context.Context, command CreateJobCommand) (domain.Job, error) {
	id, err := service.newID()
	if err != nil {
		return domain.Job{}, fmt.Errorf("generate job id: %w", err)
	}
	job, err := domain.NewJob(id, command.ProjectID, command.SourceURL, command.Operations, service.now())
	if err != nil {
		return domain.Job{}, err
	}
	if err := service.repository.Create(ctx, job); err != nil {
		return domain.Job{}, fmt.Errorf("store job: %w", err)
	}
	return job, nil
}

// Get returns a job by identifier or ErrJobNotFound.
func (service *JobService) Get(ctx context.Context, projectID, id string) (domain.Job, error) {
	return service.repository.FindByProjectID(ctx, projectID, id)
}

func randomID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "job_" + hex.EncodeToString(bytes), nil
}
