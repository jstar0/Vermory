package operatorcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"vermory/internal/provider"
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

type sourceCandidateOutput struct {
	ContinuityID      string                             `json:"continuity_id"`
	RepoRoot          string                             `json:"repo_root"`
	Disposition       runtime.SourceCandidateDisposition `json:"disposition"`
	MemoryKey         string                             `json:"memory_key"`
	TargetMemoryID    string                             `json:"target_memory_id,omitempty"`
	ObservationID     string                             `json:"observation_id"`
	CandidateMemoryID string                             `json:"candidate_memory_id,omitempty"`
	CandidateStatus   string                             `json:"candidate_status,omitempty"`
	Replayed          bool                               `json:"replayed"`
}

type sourceMatchOutput struct {
	SourceMatchID      string                             `json:"source_match_id"`
	ContinuityID       string                             `json:"continuity_id"`
	RepoRoot           string                             `json:"repo_root"`
	Decision           runtime.SourceMatchStatus          `json:"decision"`
	SelectedMemoryKey  string                             `json:"selected_memory_key,omitempty"`
	MatchedMemoryID    string                             `json:"matched_memory_id,omitempty"`
	ObservationID      string                             `json:"observation_id,omitempty"`
	CandidateMemoryID  string                             `json:"candidate_memory_id,omitempty"`
	CandidateStatus    string                             `json:"candidate_status,omitempty"`
	Disposition        runtime.SourceCandidateDisposition `json:"disposition,omitempty"`
	Provider           string                             `json:"provider"`
	Model              string                             `json:"model"`
	FailureCode        string                             `json:"failure_code,omitempty"`
	Reason             string                             `json:"reason,omitempty"`
	CandidateSetSHA256 string                             `json:"candidate_set_sha256"`
	ProviderSHA256     string                             `json:"provider_artifact_sha256,omitempty"`
	Replayed           bool                               `json:"replayed"`
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

	var sourceRoot, sourceOperationID, sourceKey, sourceContent, sourceRef string
	addSource := &cobra.Command{
		Use:   "add-source",
		Short: "Record a trusted source fact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.AddSource(cmd.Context(), sourceRoot, runtime.GovernanceWriteRequest{
					OperationID: sourceOperationID,
					MemoryKey:   sourceKey,
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
	addSource.Flags().StringVar(&sourceKey, "key", "", "stable source fact key")
	addSource.Flags().StringVar(&sourceContent, "content", "", "trusted source fact")
	addSource.Flags().StringVar(&sourceRef, "source-ref", "", "opaque source reference")
	markRequired(addSource, "repo-root", "operation-id", "content", "source-ref")

	var proposeRoot, proposeOperationID, proposeKey, proposeContent, proposeSourceRef string
	proposeSource := &cobra.Command{
		Use:   "propose-source",
		Short: "Propose a keyed source fact for operator review",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.ProposeSourceCandidate(cmd.Context(), proposeRoot, runtime.GovernanceWriteRequest{
					OperationID: proposeOperationID,
					MemoryKey:   proposeKey,
					Content:     proposeContent,
					SourceRef:   proposeSourceRef,
				})
				if err != nil {
					return err
				}
				return writeSourceCandidateJSON(cmd, service, proposeRoot, receipt)
			})
		},
	}
	proposeSource.Flags().StringVar(&proposeRoot, "repo-root", "", "absolute workspace root")
	proposeSource.Flags().StringVar(&proposeOperationID, "operation-id", "", "idempotency key")
	proposeSource.Flags().StringVar(&proposeKey, "key", "", "stable source fact key")
	proposeSource.Flags().StringVar(&proposeContent, "content", "", "candidate source fact")
	proposeSource.Flags().StringVar(&proposeSourceRef, "source-ref", "", "opaque source revision reference")
	markRequired(proposeSource, "repo-root", "operation-id", "key", "content", "source-ref")

	var matchRoot, matchOperationID, matchContent, matchSourceRef string
	var matchProvider, matchModel, matchBaseURL, matchAPIKeyEnv, matchGrokCommand string
	matchSource := &cobra.Command{
		Use:   "match-source",
		Short: "Match an unkeyed trusted source fact to the current closed set",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			llm, providerName, model, err := buildSourceMatchProvider(
				matchProvider,
				matchModel,
				matchBaseURL,
				matchAPIKeyEnv,
				matchGrokCommand,
			)
			if err != nil {
				return err
			}
			return withSourceMatching(cmd.Context(), options, llm, providerName, model, func(store *runtime.Store, service *runtime.SourceMatchingService) error {
				receipt, err := service.MatchSource(cmd.Context(), matchRoot, runtime.SourceMatchRequest{
					OperationID:   matchOperationID,
					SourceRef:     matchSourceRef,
					SourceContent: matchContent,
				})
				if err != nil {
					return err
				}
				return writeSourceMatchJSON(cmd, store, options.tenantID, matchRoot, receipt)
			})
		},
	}
	matchSource.Flags().StringVar(&matchRoot, "repo-root", "", "absolute workspace root")
	matchSource.Flags().StringVar(&matchOperationID, "operation-id", "", "idempotency key")
	matchSource.Flags().StringVar(&matchContent, "content", "", "exact trusted source fact")
	matchSource.Flags().StringVar(&matchSourceRef, "source-ref", "", "opaque source revision reference")
	matchSource.Flags().StringVar(&matchProvider, "provider", "grok-cli", "provider: grok-cli, openai-compatible, siliconflow, or duojie")
	matchSource.Flags().StringVar(&matchModel, "model", "", "provider model name")
	matchSource.Flags().StringVar(&matchBaseURL, "base-url", "", "direct provider base URL")
	matchSource.Flags().StringVar(&matchAPIKeyEnv, "api-key-env", "", "environment variable containing provider API key")
	matchSource.Flags().StringVar(&matchGrokCommand, "grok-command", "", "authenticated Grok CLI command")
	markRequired(matchSource, "repo-root", "operation-id", "content", "source-ref")

	var inspectMatchRoot, inspectMatchOperationID string
	inspectSourceMatch := &cobra.Command{
		Use:   "inspect-source-match",
		Short: "Inspect one durable source match decision",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSourceMatching(cmd.Context(), options, nil, "", "", func(store *runtime.Store, service *runtime.SourceMatchingService) error {
				receipt, err := service.InspectSourceMatch(cmd.Context(), inspectMatchRoot, inspectMatchOperationID)
				if err != nil {
					return err
				}
				return writeSourceMatchJSON(cmd, store, options.tenantID, inspectMatchRoot, receipt)
			})
		},
	}
	inspectSourceMatch.Flags().StringVar(&inspectMatchRoot, "repo-root", "", "absolute workspace root")
	inspectSourceMatch.Flags().StringVar(&inspectMatchOperationID, "operation-id", "", "source match idempotency key")
	markRequired(inspectSourceMatch, "repo-root", "operation-id")

	var acceptRoot, acceptOperationID, acceptMemoryID string
	acceptCandidate := &cobra.Command{
		Use:   "accept-candidate",
		Short: "Accept one proposed source candidate",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.AcceptCandidate(cmd.Context(), acceptRoot, acceptMemoryID, acceptOperationID)
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, acceptRoot, receipt)
			})
		},
	}
	acceptCandidate.Flags().StringVar(&acceptRoot, "repo-root", "", "absolute workspace root")
	acceptCandidate.Flags().StringVar(&acceptOperationID, "operation-id", "", "idempotency key")
	acceptCandidate.Flags().StringVar(&acceptMemoryID, "memory-id", "", "proposed source candidate")
	markRequired(acceptCandidate, "repo-root", "operation-id", "memory-id")

	var rejectRoot, rejectOperationID, rejectMemoryID string
	rejectCandidate := &cobra.Command{
		Use:   "reject-candidate",
		Short: "Reject one proposed source candidate",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.RejectCandidate(cmd.Context(), rejectRoot, rejectMemoryID, rejectOperationID)
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, rejectRoot, receipt)
			})
		},
	}
	rejectCandidate.Flags().StringVar(&rejectRoot, "repo-root", "", "absolute workspace root")
	rejectCandidate.Flags().StringVar(&rejectOperationID, "operation-id", "", "idempotency key")
	rejectCandidate.Flags().StringVar(&rejectMemoryID, "memory-id", "", "proposed source candidate")
	markRequired(rejectCandidate, "repo-root", "operation-id", "memory-id")

	var reviseSourceRoot, reviseSourceOperationID, reviseSourceMemoryID, reviseSourceContent, reviseSourceRef string
	reviseSource := &cobra.Command{
		Use:   "revise-source",
		Short: "Replace one named active fact with a trusted source revision",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withGovernance(cmd.Context(), options, func(service *runtime.GovernanceService) error {
				receipt, err := service.ReviseSource(cmd.Context(), reviseSourceRoot, reviseSourceMemoryID, runtime.GovernanceWriteRequest{
					OperationID: reviseSourceOperationID,
					Content:     reviseSourceContent,
					SourceRef:   reviseSourceRef,
				})
				if err != nil {
					return err
				}
				return writeMutationJSON(cmd, service, reviseSourceRoot, receipt)
			})
		},
	}
	reviseSource.Flags().StringVar(&reviseSourceRoot, "repo-root", "", "absolute workspace root")
	reviseSource.Flags().StringVar(&reviseSourceOperationID, "operation-id", "", "idempotency key")
	reviseSource.Flags().StringVar(&reviseSourceMemoryID, "memory-id", "", "active memory superseded by the source revision")
	reviseSource.Flags().StringVar(&reviseSourceContent, "content", "", "replacement trusted source fact")
	reviseSource.Flags().StringVar(&reviseSourceRef, "source-ref", "", "opaque replacement source reference")
	markRequired(reviseSource, "repo-root", "operation-id", "memory-id", "content", "source-ref")

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

	command.AddCommand(inspect, addSource, proposeSource, matchSource, inspectSourceMatch, acceptCandidate, rejectCandidate, reviseSource, correct, forget)
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

func withSourceMatching(
	ctx context.Context,
	options connectionOptions,
	llm provider.Provider,
	providerName string,
	model string,
	run func(*runtime.Store, *runtime.SourceMatchingService) error,
) error {
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
	return run(store, runtime.NewSourceMatchingService(store, options.tenantID, llm, providerName, model))
}

func buildSourceMatchProvider(name, model, baseURL, apiKeyEnv, grokCommand string) (provider.Provider, string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "grok-cli"
	}
	model = strings.TrimSpace(model)
	switch name {
	case "grok-cli":
		if model == "" {
			model = "grok-4.5"
		}
		return provider.NewGrokCLI(provider.GrokCLIConfig{Command: grokCommand}), name, model, nil
	case "openai-compatible", "siliconflow", "duojie":
		baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		apiKeyEnv = strings.TrimSpace(apiKeyEnv)
		switch name {
		case "siliconflow":
			if baseURL == "" {
				baseURL = "https://api.siliconflow.cn/v1"
			}
			if apiKeyEnv == "" {
				apiKeyEnv = "SILICONFLOW_API_KEY"
			}
		case "duojie":
			if baseURL == "" {
				baseURL = "https://api.duojie.games/v1"
			}
			if apiKeyEnv == "" {
				apiKeyEnv = "DUOJIE_API_KEY"
			}
		default:
			if apiKeyEnv == "" {
				apiKeyEnv = "VERMORY_PROVIDER_API_KEY"
			}
		}
		if model == "" {
			return nil, "", "", fmt.Errorf("%s provider requires --model", name)
		}
		if baseURL == "" {
			return nil, "", "", fmt.Errorf("%s provider requires --base-url", name)
		}
		apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
		if apiKey == "" {
			return nil, "", "", fmt.Errorf("%s provider requires non-empty env %s", name, apiKeyEnv)
		}
		return provider.NewOpenAICompatible(provider.Config{BaseURL: baseURL, APIKey: apiKey}), name, model, nil
	default:
		return nil, "", "", fmt.Errorf("unsupported source match provider %q", name)
	}
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

func writeSourceCandidateJSON(cmd *cobra.Command, service *runtime.GovernanceService, repoRoot string, receipt runtime.SourceCandidateReceipt) error {
	resolution, err := service.InspectWorkspace(cmd.Context(), repoRoot)
	if err != nil {
		return err
	}
	return writeJSON(cmd, sourceCandidateOutput{
		ContinuityID:      resolution.ContinuityID,
		RepoRoot:          resolution.RepoRoot,
		Disposition:       receipt.Disposition,
		MemoryKey:         receipt.MemoryKey,
		TargetMemoryID:    receipt.TargetMemoryID,
		ObservationID:     receipt.Observation.ObservationID,
		CandidateMemoryID: receipt.Candidate.MemoryID,
		CandidateStatus:   receipt.Candidate.Status,
		Replayed:          receipt.Replayed,
	})
}

func writeSourceMatchJSON(cmd *cobra.Command, store *runtime.Store, tenantID, repoRoot string, receipt runtime.SourceMatchReceipt) error {
	governance := runtime.NewGovernanceService(store, tenantID)
	resolution, memories, err := governance.ListWorkspaceMemories(cmd.Context(), repoRoot)
	if err != nil {
		return err
	}
	candidateStatus := ""
	for _, memory := range memories {
		if memory.ID == receipt.CandidateMemoryID {
			candidateStatus = memory.LifecycleStatus
			break
		}
	}
	return writeJSON(cmd, sourceMatchOutput{
		SourceMatchID:      receipt.ID,
		ContinuityID:       resolution.ContinuityID,
		RepoRoot:           resolution.RepoRoot,
		Decision:           receipt.Status,
		SelectedMemoryKey:  receipt.SelectedMemoryKey,
		MatchedMemoryID:    receipt.TargetMemoryID,
		ObservationID:      receipt.ObservationID,
		CandidateMemoryID:  receipt.CandidateMemoryID,
		CandidateStatus:    candidateStatus,
		Disposition:        receipt.Disposition,
		Provider:           receipt.ProviderName,
		Model:              receipt.ResolvedModel,
		FailureCode:        receipt.FailureCode,
		Reason:             receipt.Reason,
		CandidateSetSHA256: receipt.CandidateSetFingerprint,
		ProviderSHA256:     receipt.ProviderArtifactSHA256,
		Replayed:           receipt.Replayed,
	})
}
