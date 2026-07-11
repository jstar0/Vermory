package domain

type ContinuityLine string

const (
	ContinuityLineWorkspace      ContinuityLine = "workspace"
	ContinuityLineConversation   ContinuityLine = "conversation"
	ContinuityLineGlobalDefaults ContinuityLine = "global_defaults"
)

type BridgeAction string

const (
	BridgeActionPromote BridgeAction = "promote"
	BridgeActionLink    BridgeAction = "link"
	BridgeActionExport  BridgeAction = "export"
	BridgeActionAdopt   BridgeAction = "adopt"
	BridgeActionRebind  BridgeAction = "rebind"
)

type BenchmarkName string

type BenchmarkCapability string

type AuditAction string

type AnchorStrength string

const (
	AnchorStrengthStrong AnchorStrength = "strong"
	AnchorStrengthWeak   AnchorStrength = "weak"
)

type TargetProfileID string
