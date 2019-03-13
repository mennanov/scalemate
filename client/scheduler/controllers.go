package scheduler

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
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
	flags *CreateJobCmdFlags,
) (*scheduler_proto.Job, error) {
	job, err := flags.ToProto()
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
	flags *ListJobsCmdFlags,
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
	request := flags.ToProto()

	// Set the Username value to the currently authenticated username.
	request.Username = claims.Username

	return schedulerClient.ListJobs(context.Background(), request, grpc.PerRPCCredentials(jwtCredentials))
}

// CancelJobController cancels an existing Job by its ID.
func CancelJobController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	jobID uint64,
) (*scheduler_proto.Job, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.CancelJob(
		context.Background(),
		&scheduler_proto.JobLookupRequest{JobId: jobID},
		grpc.PerRPCCredentials(jwtCredentials))
}

// GetTaskController gets an existing Task by its ID.
func GetTaskController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	taskID uint64,
) (*scheduler_proto.Task, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.GetTask(
		context.Background(),
		&scheduler_proto.TaskLookupRequest{TaskId: taskID},
		grpc.PerRPCCredentials(jwtCredentials))
}

// ListTasksController lists the Tasks that satisfy the criteria for the currently authenticated user.
func ListTasksController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	jobIDs []uint64,
	flags *ListTasksCmdFlags,
) (*scheduler_proto.ListTasksResponse, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	claims, err := auth.NewClaimsFromString(tokens.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, newClaimsFromStringFailedMsg)
	}
	request := flags.ToProto()
	request.JobId = jobIDs

	// Set the Username value to the currently authenticated username.
	request.Username = claims.Username
	return schedulerClient.ListTasks(context.Background(), request, grpc.PerRPCCredentials(jwtCredentials))
}

// CancelTaskController cancels an existing Task by its ID.
func CancelTaskController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	taskID uint64,
) (*scheduler_proto.Task, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.CancelTask(
		context.Background(),
		&scheduler_proto.TaskLookupRequest{TaskId: taskID},
		grpc.PerRPCCredentials(jwtCredentials))
}

// IterateTasksController receives a stream of Tasks for the given Job ID.
func IterateTasksController(
	accountsClient accounts_proto.AccountsClient,
	schedulerClient scheduler_proto.SchedulerClient,
	jobID uint64,
	includeExisting bool,
) (scheduler_proto.Scheduler_IterateTasksClient, error) {
	tokens, err := auth.LoadTokens()
	if err != nil {
		return nil, errors.Wrap(err, loadTokensFailedErrMsg)
	}
	jwtCredentials := auth.NewJWTCredentials(accountsClient, tokens, auth.SaveTokens)

	return schedulerClient.IterateTasks(
		context.Background(),
		&scheduler_proto.IterateTasksRequest{JobId: jobID, IncludeExisting: includeExisting},
		grpc.PerRPCCredentials(jwtCredentials))
}

// GetNodeController gets an existing Node by its ID.
func GetNodeController(
	schedulerClient scheduler_proto.SchedulerClient,
	nodeID uint64,
) (*scheduler_proto.Node, error) {
	return schedulerClient.GetNode(
		context.Background(),
		&scheduler_proto.NodeLookupRequest{NodeId: nodeID})
}

// ListNodesController lists the Nodes that satisfy the criteria.
func ListNodesController(
	schedulerClient scheduler_proto.SchedulerClient,
	flags *ListNodesCmdFlags,
) (*scheduler_proto.ListNodesResponse, error) {
	r := flags.ToProto()
	return schedulerClient.ListNodes(context.Background(), r)
}

// ListCpuModelsController lists aggregated CPU models for the currently online Nodes.
func ListCpuModelsController(
	schedulerClient scheduler_proto.SchedulerClient,
	class scheduler_proto.CPUClass,
) (*scheduler_proto.ListCpuModelsResponse, error) {
	return schedulerClient.ListCpuModels(context.Background(), &scheduler_proto.ListCpuModelsRequest{
		CpuClass: class,
	})
}

// ListGpuModelsController lists aggregated GPU models for the currently online Nodes.
func ListGpuModelsController(
	schedulerClient scheduler_proto.SchedulerClient,
	class scheduler_proto.GPUClass,
) (*scheduler_proto.ListGpuModelsResponse, error) {
	return schedulerClient.ListGpuModels(context.Background(), &scheduler_proto.ListGpuModelsRequest{
		GpuClass: class,
	})
}

// ListDiskModelsController lists aggregated Disk models for the currently online Nodes.
func ListDiskModelsController(
	schedulerClient scheduler_proto.SchedulerClient,
	class scheduler_proto.DiskClass,
) (*scheduler_proto.ListDiskModelsResponse, error) {
	return schedulerClient.ListDiskModels(context.Background(), &scheduler_proto.ListDiskModelsRequest{
		DiskClass: class,
	})
}

// ListMemoryModelsController lists aggregated Memory models for the currently online Nodes.
func ListMemoryModelsController(
	schedulerClient scheduler_proto.SchedulerClient,
) (*scheduler_proto.ListMemoryModelsResponse, error) {
	return schedulerClient.ListMemoryModels(context.Background(), &empty.Empty{})
}
