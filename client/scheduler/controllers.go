package scheduler

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
)

// ServiceAddr is a network address of the Scheduler service.
var ServiceAddr string

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
		return nil, errors.Wrap(err, "failed to load authentication tokens. Are you logged in?")
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	claims, err := auth.NewClaimsFromString(tokens.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse authentication tokens. Try re-login")
	}
	job.Username = claims.Username

	return schedulerClient.CreateJob(context.Background(), job, grpc.PerRPCCredentials(jwtCredentials))
}
