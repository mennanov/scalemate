package scheduler

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
)

var (
	loadTokensFailedErrMsg       = "failed to load authentication tokens. Are you logged in?"
	newClaimsFromStringFailedMsg = "failed to parse authentication tokens. Try re-login"
)

// CreateJobController creates a new Job entity by calling Scheduler.CreateJob method.
func CreateJobController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	image string,
	command string,
	flags *JobsCreateCmdFlags,
) (*scheduler_proto.Job, error) {
	job, err := flags.ToJobProto()
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse flags")
	}
	if image == "" {
		return nil, errors.New("image is an empty string")
	}
	job.RunConfig.Image = image
	job.RunConfig.Command = command

	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	claims, err := auth.NewClaimsFromString(tokens.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, newClaimsFromStringFailedMsg)
	}
	job.Username = claims.Username

	return schedulerClient.CreateJob(context.Background(), job, grpc.PerRPCCredentials(jwtCredentials))
}

// GetJobController gets an existing Job by its ID.
func GetJobController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	jobID uint64,
) (*scheduler_proto.Job, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.GetJob(
		context.Background(),
		&scheduler_proto.JobLookupRequest{JobId: jobID},
		grpc.PerRPCCredentials(jwtCredentials))
}

// ListJobsController lists the Jobs that satisfy the criteria for the currently authenticated user.
func ListJobsController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	flags *JobsListCmdFlags,
) (*scheduler_proto.ListJobsResponse, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	claims, err := auth.NewClaimsFromString(tokens.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, newClaimsFromStringFailedMsg)
	}
	request := flags.ToListJobsRequestProto()

	// Set the Username value to the currently authenticated username.
	request.Username = claims.Username

	return schedulerClient.ListJobs(context.Background(), request, grpc.PerRPCCredentials(jwtCredentials))
}
