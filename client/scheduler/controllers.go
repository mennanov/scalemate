package scheduler

import (
	"context"

	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/scheduler/scheduler_proto"
	"google.golang.org/grpc"

	"github.com/mennanov/scalemate/shared/auth"
)

// ServiceAddr is a network address of the Scheduler service.
var ServiceAddr string

// CreateJobController creates a new Job entity by calling Scheduler.CreateJob method.
func CreateJobController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	job *scheduler_proto.Job,
) (*scheduler_proto.Job, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, err
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.CreateJob(context.Background(), job, grpc.PerRPCCredentials(jwtCredentials))
}
