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

func NewDefaultsCommand() *cobra.Command {
	options := connectionOptions{}
	command := &cobra.Command{
		Use:   "defaults",
		Short: "Manage explicit global defaults",
	}
	addConnectionFlags(command, &options)

	inspect := &cobra.Command{
		Use:   "inspect",
		Short: "List global default lifecycle state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGlobalDefaults(cmd.Context(), options, func(service *runtime.GlobalDefaultsService) error {
				inspection, err := service.Inspect(cmd.Context())
				if err != nil {
					return err
				}
				return writeJSON(cmd, inspection)
			})
		},
	}

	var setOperationID, setKey, setContent string
	set := &cobra.Command{
		Use:   "set",
		Short: "Create one explicit global default",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGlobalDefaults(cmd.Context(), options, func(service *runtime.GlobalDefaultsService) error {
				receipt, err := service.Set(cmd.Context(), runtime.SetGlobalDefaultRequest{
					OperationID: setOperationID,
					Key:         setKey,
					Content:     setContent,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	set.Flags().StringVar(&setOperationID, "operation-id", "", "idempotency key")
	set.Flags().StringVar(&setKey, "key", "", "stable lowercase default key")
	set.Flags().StringVar(&setContent, "content", "", "semantic default content")
	markRequired(set, "operation-id", "key", "content")

	var correctOperationID, correctMemoryID, correctContent string
	correct := &cobra.Command{
		Use:   "correct",
		Short: "Replace one named active global default",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGlobalDefaults(cmd.Context(), options, func(service *runtime.GlobalDefaultsService) error {
				receipt, err := service.Correct(cmd.Context(), runtime.CorrectGlobalDefaultRequest{
					OperationID: correctOperationID,
					MemoryID:    correctMemoryID,
					Content:     correctContent,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	correct.Flags().StringVar(&correctOperationID, "operation-id", "", "idempotency key")
	correct.Flags().StringVar(&correctMemoryID, "memory-id", "", "active global default to replace")
	correct.Flags().StringVar(&correctContent, "content", "", "replacement semantic content")
	markRequired(correct, "operation-id", "memory-id", "content")

	var forgetOperationID, forgetMemoryID string
	forget := &cobra.Command{
		Use:   "forget",
		Short: "Delete one named global default",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGlobalDefaults(cmd.Context(), options, func(service *runtime.GlobalDefaultsService) error {
				receipt, err := service.Forget(cmd.Context(), runtime.ForgetGlobalDefaultRequest{
					OperationID: forgetOperationID,
					MemoryID:    forgetMemoryID,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	forget.Flags().StringVar(&forgetOperationID, "operation-id", "", "idempotency key")
	forget.Flags().StringVar(&forgetMemoryID, "memory-id", "", "global default to delete")
	markRequired(forget, "operation-id", "memory-id")

	command.AddCommand(inspect, set, correct, forget)
	return command
}

func NewBridgeCommand() *cobra.Command {
	options := connectionOptions{}
	command := &cobra.Command{Use: "bridge", Short: "Manage explicit cross-continuity bridges"}
	addConnectionFlags(command, &options)

	var promoteOperationID, promoteChannel, promoteThreadID, promoteRepoRoot string
	var promoteMemoryIDs []string
	promote := &cobra.Command{
		Use: "promote", Short: "Promote selected conversation memory into a workspace", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.PromoteConversationToWorkspace(cmd.Context(), runtime.PromoteConversationToWorkspaceRequest{
					OperationID:    promoteOperationID,
					Source:         runtime.ConversationAnchor{Channel: promoteChannel, ThreadID: promoteThreadID},
					TargetRepoRoot: promoteRepoRoot,
					MemoryIDs:      promoteMemoryIDs,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	promote.Flags().StringVar(&promoteOperationID, "operation-id", "", "idempotency key")
	promote.Flags().StringVar(&promoteChannel, "source-channel", "", "source conversation channel")
	promote.Flags().StringVar(&promoteThreadID, "source-thread-id", "", "source conversation thread")
	promote.Flags().StringVar(&promoteRepoRoot, "target-repo-root", "", "confirmed target workspace root")
	promote.Flags().StringSliceVar(&promoteMemoryIDs, "memory-id", nil, "selected active source memory id")
	markRequired(promote, "operation-id", "source-channel", "source-thread-id", "target-repo-root", "memory-id")

	var linkOperationID, primaryChannel, primaryThreadID, linkedChannel, linkedThreadID string
	link := &cobra.Command{
		Use: "link", Short: "Link governed memory across two conversation anchors", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.LinkConversations(cmd.Context(), runtime.LinkConversationsRequest{
					OperationID: linkOperationID,
					Primary:     runtime.ConversationAnchor{Channel: primaryChannel, ThreadID: primaryThreadID},
					Linked:      runtime.ConversationAnchor{Channel: linkedChannel, ThreadID: linkedThreadID},
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	link.Flags().StringVar(&linkOperationID, "operation-id", "", "idempotency key")
	link.Flags().StringVar(&primaryChannel, "primary-channel", "", "primary conversation channel")
	link.Flags().StringVar(&primaryThreadID, "primary-thread-id", "", "primary conversation thread")
	link.Flags().StringVar(&linkedChannel, "linked-channel", "", "linked conversation channel")
	link.Flags().StringVar(&linkedThreadID, "linked-thread-id", "", "linked conversation thread")
	markRequired(link, "operation-id", "primary-channel", "primary-thread-id", "linked-channel", "linked-thread-id")

	var exportOperationID, exportRepoRoot, exportTitle, exportProfile string
	var exportMemoryIDs []string
	export := &cobra.Command{
		Use: "export", Short: "Export a bounded workspace memory view", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.ExportWorkspace(cmd.Context(), runtime.ExportWorkspaceRequest{
					OperationID: exportOperationID, RepoRoot: exportRepoRoot, MemoryIDs: exportMemoryIDs,
					Title: exportTitle, TargetProfile: exportProfile,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	export.Flags().StringVar(&exportOperationID, "operation-id", "", "idempotency key")
	export.Flags().StringVar(&exportRepoRoot, "repo-root", "", "confirmed workspace root")
	export.Flags().StringSliceVar(&exportMemoryIDs, "memory-id", nil, "selected active workspace memory id")
	export.Flags().StringVar(&exportTitle, "title", "", "export title")
	export.Flags().StringVar(&exportProfile, "target-profile", "", "target consumer profile")
	markRequired(export, "operation-id", "repo-root", "memory-id", "title", "target-profile")

	var adoptOperationID, adoptExistingRoot, adoptNewRoot string
	adopt := &cobra.Command{
		Use: "adopt", Short: "Add a confirmed alias to an existing workspace", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.AdoptWorkspaceAnchor(cmd.Context(), runtime.AdoptWorkspaceAnchorRequest{
					OperationID: adoptOperationID, ExistingRepoRoot: adoptExistingRoot, NewRepoRoot: adoptNewRoot,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	adopt.Flags().StringVar(&adoptOperationID, "operation-id", "", "idempotency key")
	adopt.Flags().StringVar(&adoptExistingRoot, "existing-repo-root", "", "existing confirmed workspace root")
	adopt.Flags().StringVar(&adoptNewRoot, "new-repo-root", "", "new alias root")
	markRequired(adopt, "operation-id", "existing-repo-root", "new-repo-root")

	var rebindOperationID, rebindOldRoot, rebindNewRoot string
	rebind := &cobra.Command{
		Use: "rebind", Short: "Move a workspace continuity to a new root", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.RebindWorkspace(cmd.Context(), runtime.RebindWorkspaceRequest{
					OperationID: rebindOperationID, OldRepoRoot: rebindOldRoot, NewRepoRoot: rebindNewRoot,
				})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	rebind.Flags().StringVar(&rebindOperationID, "operation-id", "", "idempotency key")
	rebind.Flags().StringVar(&rebindOldRoot, "old-repo-root", "", "current confirmed workspace root")
	rebind.Flags().StringVar(&rebindNewRoot, "new-repo-root", "", "replacement workspace root")
	markRequired(rebind, "operation-id", "old-repo-root", "new-repo-root")

	var reverseOperationID, reverseBridgeID string
	reverse := &cobra.Command{
		Use: "reverse", Short: "Reverse or revoke one bridge", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.Reverse(cmd.Context(), runtime.ReverseBridgeRequest{OperationID: reverseOperationID, BridgeID: reverseBridgeID})
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	reverse.Flags().StringVar(&reverseOperationID, "operation-id", "", "idempotency key")
	reverse.Flags().StringVar(&reverseBridgeID, "bridge-id", "", "bridge to reverse or revoke")
	markRequired(reverse, "operation-id", "bridge-id")

	var inspectBridgeID string
	inspect := &cobra.Command{
		Use: "inspect", Short: "Inspect one durable bridge", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withBridges(cmd.Context(), options, func(service *runtime.BridgeService) error {
				receipt, err := service.Inspect(cmd.Context(), inspectBridgeID)
				if err != nil {
					return err
				}
				return writeJSON(cmd, receipt)
			})
		},
	}
	inspect.Flags().StringVar(&inspectBridgeID, "bridge-id", "", "bridge to inspect")
	markRequired(inspect, "bridge-id")

	command.AddCommand(promote, link, export, adopt, rebind, reverse, inspect)
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

func withGlobalDefaults(ctx context.Context, options connectionOptions, run func(*runtime.GlobalDefaultsService) error) error {
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
	return run(runtime.NewGlobalDefaultsService(store, options.tenantID))
}

func withBridges(ctx context.Context, options connectionOptions, run func(*runtime.BridgeService) error) error {
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
	return run(runtime.NewBridgeService(store, options.tenantID))
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
