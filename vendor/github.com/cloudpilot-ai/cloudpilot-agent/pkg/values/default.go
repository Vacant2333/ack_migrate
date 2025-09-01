package values

const (
	CloudPilotNamespace        = "cloudpilot"
	CloudPilotNamespaceEnv     = "CLOUDPILOT_NAMESPACE"
	CloudPilotResourceLockName = "cloudpilot"

	CloudProviderAWS          = "aws"
	CloudProviderGCE          = "gce"
	CloudProviderAzure        = "azure"
	CloudProviderAlibabaCloud = "alibabacloud"

	OnDemandCapacityType = "ON_DEMAND"
	SpotCapacityType     = "SPOT"

	LeaderElectionIDEnv           = "LEADER_ELECTION_ID"
	CloudPilotAPIKeyEnv           = "CLOUDPILOT_API_KEY"
	CloudPilotAPIEndpointEnv      = "CLOUDPILOT_API_ENDPOINT"
	CloudProviderEnv              = "CLOUD_PROVIDER"
	RegionEnv                     = "REGION"
	ClusterIDEnv                  = "CLUSTER_ID"
	WebhookNameEnv                = "WEBHOOK_NAME"
	WebhookNamespaceEnv           = "WEBHOOK_NAMESPACE"
	PriceAPIEndpointEnv           = "PRICE_API_ENDPOINT"
	PredictorAPIEndpoint          = "PREDICTOR_API_ENDPOINT"
	ClusterNameEnv                = "CLUSTER_NAME"
	AlibabaCloudAK                = "ALIBABACLOUD_AK"
	AlibabaCloudSK                = "ALIBABACLOUD_SK"
	CloudPilotDebugClusterEnv     = "CLOUDPILOT_DEBUG_CLUSTER"
	CloudPilotDebugClusterNameEnv = "CLOUDPILOT_DEBUG_CLUSTER_NAME"
	EnableNodeRepairEnv           = "ENABLE_NODE_REPAIR"
	DisableWorkloadUploadingEnv   = "DISABLE_WORKLOAD_UPLOADING"

	CloudPilotNodeDrainTaintKey                   = "node.cloudpilot.ai/drain"
	CloudPilotNodeDrainTaintEffect                = "NoSchedule"
	CloudPilotManagedNodeLabelKey                 = "node.cloudpilot.ai/managed"
	CloudPilotNodeDrainLabelKey                   = "node.cloudpilot.ai/drain"
	CloudPilotNodeDrainReasonInterruptLabelValue  = "interrupt"
	CloudPilotNodeDrainReasonRebalanceLabelValue  = "provider"
	CloudPilotNodeDrainReasonProactiveLabelValue  = "proactive"
	CloudPilotNodeDrainReasonPredictionLabelValue = "prediction"
	CloudPilotMinNonSpotReplicasLabelKey          = "workload.cloudpilot.ai/min-nonspot"
	CloudPilotSpotFriendlyLabelKey                = "workload.cloudpilot.ai/spot-friendly"
	CloudPilotWorkloadMinNodesLabelKey            = "workload.cloudpilot.ai/min-nodes"
	CloudPilotSpotFriendlyLabelValueFalse         = "false"

	CloudPilotNodeSafeToTerminateLabelKey = "node.cloudpilot.ai/safe-to-terminate"

	AWSNodeCapacityTypeLabelKey = "eks.amazonaws.com/capacityType"

	CloudPilotNodePoolEnableLabelKey                = "cloudpilot.ai/nodepool-enable"
	CloudPilotDisablePodAntiAffinityLabelKey        = "cloudpilot.ai/anti-affinity-disable"
	CloudPilotDiversityInstanceTypeLabelKey         = "cloudpilot.ai/diversity-instancetype"
	CloudPilotWorkloadDiversityInstanceTypeLabelKey = "workload.cloudpilot.ai/diversity-instancetype"

	CloudPilotKarpenterRolloutAt = "karpenter.cloudpilot.ai/rolloutAt"

	OptimizerAlibabaCloudDeploymentName = "cloudpilot-alibabacloud-optimizer"
	OptimizerAWSDeploymentName          = "cloudpilot-aws-optimizer"
	ControllerDeploymentName            = "cloudpilot-controller"

	RemoteClusterEnv = "REMOTE_CLUSTER"

	DefaultFileTimeout = "2232h"
)

var AllowedClusterProvider = []string{
	CloudProviderAWS,
	CloudProviderAlibabaCloud,
}
