package operatorcli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"vermory/internal/runtime"

	"github.com/spf13/cobra"
)

type connectionOptions struct {
	databaseURL string
	tenantID    string
}

type workspaceOutput struct {
	Status       string `json:"status"`
	ContinuityID string `json:"continuity_id,omitempty"`
	RepoRoot     string `json:"repo_root"`
}

type mutationOutput struct {
	ContinuityID  string `json:"continuity_id"`
	ObservationID string `json:"observation_id"`
	MemoryID      string `json:"memory_id"`
	MemoryStatus  string `json:"memory_status"`
	Replayed      bool   `json:"replayed"`
}

type memoryListOutput struct {
	ContinuityID string                   `json:"continuity_id"`
	RepoRoot     string                   `json:"repo_root"`
	Memories     []runtime.GovernedMemory `json:"memories"`
}

func NewWorkspaceCommand() *cobra.Command {
	options := connectionOptions{}
	command := &cobra.Command{
		Use:   "workspace",
		Short: "Manage trusted workspace bindings",
	}
	addConnectionFlags(command, &options)

	var confirmRoot string
	confirm := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm a workspace binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				resolution, err := service.ConfirmWorkspace(cmd.Context(), confirmRoot)
				if err != nil {
					return err
				}
				return writeJSON(cmd, workspaceOutput{
					Status:       string(resolution.Status),
					ContinuityID: resolution.ContinuityID,
					RepoRoot:     resolution.RepoRoot,
				})
			})
		},
	}
	confirm.Flags().StringVar(&confirmRoot, "repo-root", "", "absolute workspace root")
	_ = confirm.MarkFlagRequired("repo-root")

	var inspectRoot string
	inspect := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect a workspace binding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				resolution, err := service.InspectWorkspace(cmd.Context(), inspectRoot)
				if err != nil {
					return err
				}
				return writeJSON(cmd, workspaceOutput{
					Status:       string(resolution.Status),
					ContinuityID: resolution.ContinuityID,
					RepoRoot:     resolution.RepoRoot,
				})
			})
		},
	}
	inspect.Flags().StringVar(&inspectRoot, "repo-root", "", "absolute workspace root")
	_ = inspect.MarkFlagRequired("repo-root")

	command.AddCommand(confirm, inspect)
	return command
}

func NewMemoryCommand() *cobra.Command {
	options := connectionOptions{}
	command := &cobra.Command{
		Use:   "memory",
		Short: "Apply explicit local memory governance",
	}
	addConnectionFlags(command, &options)

	var inspectRoot string
	inspect := &cobra.Command{
		Use:   "inspect",
		Short: "List governed memories for a confirmed workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				resolution, memories, err := service.ListWorkspaceMemories(cmd.Context(), inspectRoot)
				if err != nil {
					return err
				}
				return writeJSON(cmd, memoryListOutput{
					ContinuityID: resolution.ContinuityID,
					RepoRoot:     resolution.RepoRoot,
					Memories:     memories,
				})
			})
		},
	}
	inspect.Flags().StringVar(&inspectRoot, "repo-root", "", "absolute workspace root")
	_ = inspect.MarkFlagRequired("repo-root")

	var sourceRoot, sourceOperationID, sourceContent, sourceRef string
	addSource := &cobra.Command{
		Use:   "add-source",
		Short: "Record a trusted source fact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.AddSource(cmd.Context(), sourceRoot, runtime.GovernanceWriteRequest{
					OperationID: sourceOperationID,
					Content:     sourceContent,
					SourceRef:   sourceRef,
				})
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, sourceRoot, receipt)
			})
		},
	}
	addSource.Flags().StringVar(&sourceRoot, "repo-root", "", "absolute workspace root")
	addSource.Flags().StringVar(&sourceOperationID, "operation-id", "", "idempotency key")
	addSource.Flags().StringVar(&sourceContent, "content", "", "trusted source fact")
	addSource.Flags().StringVar(&sourceRef, "source-ref", "", "opaque source reference")
	markRequired(addSource, "repo-root", "operation-id", "content", "source-ref")

	var correctRoot, correctOperationID, correctMemoryID, correctContent string
	correct := &cobra.Command{
		Use:   "correct",
		Short: "Replace one named active fact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.Correct(cmd.Context(), correctRoot, correctMemoryID, runtime.GovernanceWriteRequest{
					OperationID: correctOperationID,
					Content:     correctContent,
				})
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, correctRoot, receipt)
			})
		},
	}
	correct.Flags().StringVar(&correctRoot, "repo-root", "", "absolute workspace root")
	correct.Flags().StringVar(&correctOperationID, "operation-id", "", "idempotency key")
	correct.Flags().StringVar(&correctMemoryID, "memory-id", "", "active memory to supersede")
	correct.Flags().StringVar(&correctContent, "content", "", "replacement fact")
	markRequired(correct, "repo-root", "operation-id", "memory-id", "content")

	var forgetRoot, forgetOperationID, forgetMemoryID string
	forget := &cobra.Command{
		Use:   "forget",
		Short: "Redact one named memory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.Forget(cmd.Context(), forgetRoot, forgetMemoryID, forgetOperationID)
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, forgetRoot, receipt)
			})
		},
	}
	forget.Flags().StringVar(&forgetRoot, "repo-root", "", "absolute workspace root")
	forget.Flags().StringVar(&forgetOperationID, "operation-id", "", "idempotency key")
	forget.Flags().StringVar(&forgetMemoryID, "memory-id", "", "memory to redact")
	markRequired(forget, "repo-root", "operation-id", "memory-id")

	command.AddCommand(inspect, addSource, correct, forget)
	return command
}

func addConnectionFlags(command *cobra.Command, options *connectionOptions) {
	command.PersistentFlags().StringVar(&options.databaseURL, "database-url", "", "PostgreSQL connection URL")
	command.PersistentFlags().StringVar(&options.tenantID, "tenant-id", "", "server-owned tenant identifier")
	_ = command.MarkPersistentFlagRequired("database-url")
	_ = command.MarkPersistentFlagRequired("tenant-id")
}

func markRequired(command *cobra.Command, names ...string) {
	for _, name := range names {
		_ = command.MarkFlagRequired(name)
	}
}

func withGovernance(ctx context.Context, options connectionOptions, run func(*runtime.GovernanceService) error) error {
	if strings.TrimSpace(options.databaseURL) == "" {
		return fmt.Errorf("--database-url is required")
	}
	if strings.TrimSpace(options.tenantID) == "" {
		return fmt.Errorf("--tenant-id is required")
	}
	store, err := runtime.OpenStore(ctx, options.databaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	return run(runtime.NewGovernanceService(store, options.tenantID))
}

func writeJSON(cmd *cobra.Command, value any) error {
	return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
}

func writeMutationJSON(cmd *cobra.Command, service *runtime.GovernanceService, repoRoot string, receipt runtime.GovernedObservationReceipt) error {
	resolution, err := service.InspectWorkspace(cmd.Context(), repoRoot)
	if err != nil {
		return err
	}
	return writeJSON(cmd, mutationOutput{
		ContinuityID:  resolution.ContinuityID,
		ObservationID: receipt.Observation.ObservationID,
		MemoryID:      receipt.Memory.MemoryID,
		MemoryStatus:  receipt.Memory.Status,
		Replayed:      receipt.Observation.Replayed || receipt.Memory.Replayed,
	})
}
