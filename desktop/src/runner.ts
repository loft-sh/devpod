import { V1Affinity } from "@loft-enterprise/client/gen/models/V1Affinity"
import { V1Container } from "@loft-enterprise/client/gen/models/V1Container"
import { V1EnvFromSource } from "@loft-enterprise/client/gen/models/V1EnvFromSource"
import { V1EnvVar } from "@loft-enterprise/client/gen/models/V1EnvVar"
import { V1HostAlias } from "@loft-enterprise/client/gen/models/V1HostAlias"
import { V1ObjectMeta } from "@loft-enterprise/client/gen/models/V1ObjectMeta"
import { V1ResourceRequirements } from "@loft-enterprise/client/gen/models/V1ResourceRequirements"
import { V1Toleration } from "@loft-enterprise/client/gen/models/V1Toleration"
import { V1Volume } from "@loft-enterprise/client/gen/models/V1Volume"
import { V1VolumeMount } from "@loft-enterprise/client/gen/models/V1VolumeMount"
import { StorageV1Condition } from "@loft-enterprise/client/gen/models/agentstorageV1Condition"
import { StorageV1Access } from "@loft-enterprise/client/gen/models/storageV1Access"
import { StorageV1TemplateMetadata } from "@loft-enterprise/client/gen/models/storageV1TemplateMetadata"
import { StorageV1UserOrTeam } from "@loft-enterprise/client/gen/models/storageV1UserOrTeam"

/**
 * Runner holds the Runner information
 */
export class ManagementV1Runner {
  /**
   * APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
   */
  "apiVersion"?: string
  /**
   * Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
   */
  "kind"?: string
  "metadata"?: V1ObjectMeta
  "spec"?: ManagementV1RunnerSpec
  "status"?: ManagementV1RunnerStatus

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "apiVersion",
      baseName: "apiVersion",
      type: "string",
      format: "",
    },
    {
      name: "kind",
      baseName: "kind",
      type: "string",
      format: "",
    },
    {
      name: "metadata",
      baseName: "metadata",
      type: "V1ObjectMeta",
      format: "",
    },
    {
      name: "spec",
      baseName: "spec",
      type: "ManagementV1RunnerSpec",
      format: "",
    },
    {
      name: "status",
      baseName: "status",
      type: "ManagementV1RunnerStatus",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return ManagementV1Runner.attributeTypeMap
  }

  public constructor() {}
}

/**
 * RunnerSpec holds the specification
 */
export class ManagementV1RunnerSpec {
  /**
   * Access holds the access rights for users and teams
   */
  "access"?: Array<StorageV1Access>
  "clusterRef"?: StorageV1RunnerClusterRef
  /**
   * Description describes a cluster access object
   */
  "description"?: string
  /**
   * The display name shown in the UI
   */
  "displayName"?: string
  /**
   * Endpoint is the hostname used to connect directly to the runner
   */
  "endpoint"?: string
  /**
   * NetworkPeerName is the network peer name used to connect directly to the runner
   */
  "networkPeerName"?: string
  "owner"?: StorageV1UserOrTeam
  /**
   * If unusable is true, no DevPod workspaces can be scheduled on this runner.
   */
  "unusable"?: boolean

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "access",
      baseName: "access",
      type: "Array<StorageV1Access>",
      format: "",
    },
    {
      name: "clusterRef",
      baseName: "clusterRef",
      type: "StorageV1RunnerClusterRef",
      format: "",
    },
    {
      name: "description",
      baseName: "description",
      type: "string",
      format: "",
    },
    {
      name: "displayName",
      baseName: "displayName",
      type: "string",
      format: "",
    },
    {
      name: "endpoint",
      baseName: "endpoint",
      type: "string",
      format: "",
    },
    {
      name: "networkPeerName",
      baseName: "networkPeerName",
      type: "string",
      format: "",
    },
    {
      name: "owner",
      baseName: "owner",
      type: "StorageV1UserOrTeam",
      format: "",
    },
    {
      name: "unusable",
      baseName: "unusable",
      type: "boolean",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return ManagementV1RunnerSpec.attributeTypeMap
  }

  public constructor() {}
}

/**
 * RunnerStatus holds the status
 */
export class ManagementV1RunnerStatus {
  /**
   * Conditions holds several conditions the virtual cluster might be in
   */
  "conditions"?: Array<StorageV1Condition>
  /**
   * Message describes the reason in human-readable form
   */
  "message"?: string
  /**
   * Phase describes the current phase the space instance is in
   */
  "phase"?: string
  /**
   * Reason describes the reason in machine-readable form
   */
  "reason"?: string

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "conditions",
      baseName: "conditions",
      type: "Array<StorageV1Condition>",
      format: "",
    },
    {
      name: "message",
      baseName: "message",
      type: "string",
      format: "",
    },
    {
      name: "phase",
      baseName: "phase",
      type: "string",
      format: "",
    },
    {
      name: "reason",
      baseName: "reason",
      type: "string",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return ManagementV1RunnerStatus.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerClusterRef {
  /**
   * Cluster is the connected cluster the space will be created in
   */
  "cluster"?: string
  /**
   * Namespace is the namespace inside the connected cluster holding the space
   */
  "namespace"?: string
  "persistentVolumeClaimTemplate"?: StorageV1RunnerPersistentVolumeClaimTemplate
  "podTemplate"?: StorageV1RunnerPodTemplate
  "serviceTemplate"?: StorageV1RunnerServiceTemplate

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "cluster",
      baseName: "cluster",
      type: "string",
      format: "",
    },
    {
      name: "namespace",
      baseName: "namespace",
      type: "string",
      format: "",
    },
    {
      name: "persistentVolumeClaimTemplate",
      baseName: "persistentVolumeClaimTemplate",
      type: "StorageV1RunnerPersistentVolumeClaimTemplate",
      format: "",
    },
    {
      name: "podTemplate",
      baseName: "podTemplate",
      type: "StorageV1RunnerPodTemplate",
      format: "",
    },
    {
      name: "serviceTemplate",
      baseName: "serviceTemplate",
      type: "StorageV1RunnerServiceTemplate",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerClusterRef.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerPodTemplate {
  "metadata"?: StorageV1TemplateMetadata
  "spec"?: StorageV1RunnerPodTemplateSpec

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "metadata",
      baseName: "metadata",
      type: "StorageV1TemplateMetadata",
      format: "",
    },
    {
      name: "spec",
      baseName: "spec",
      type: "StorageV1RunnerPodTemplateSpec",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerPodTemplate.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerPersistentVolumeClaimTemplateSpec {
  /**
   * accessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
   */
  "accessModes"?: Array<StorageV1RunnerPersistentVolumeClaimTemplateSpecAccessModesEnum>
  /**
   * storageClassName is the name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
   */
  "storageClassName"?: string
  /**
   * storageSize is the size of the storage to reserve for the pvc
   */
  "storageSize"?: string

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "accessModes",
      baseName: "accessModes",
      type: "Array<StorageV1RunnerPersistentVolumeClaimTemplateSpecAccessModesEnum>",
      format: "",
    },
    {
      name: "storageClassName",
      baseName: "storageClassName",
      type: "string",
      format: "",
    },
    {
      name: "storageSize",
      baseName: "storageSize",
      type: "string",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerPersistentVolumeClaimTemplateSpec.attributeTypeMap
  }

  public constructor() {}
}

export enum StorageV1RunnerPersistentVolumeClaimTemplateSpecAccessModesEnum {
  ReadOnlyMany = "ReadOnlyMany",
  ReadWriteMany = "ReadWriteMany",
  ReadWriteOnce = "ReadWriteOnce",
  ReadWriteOncePod = "ReadWriteOncePod",
}

export class StorageV1RunnerPodTemplateSpec {
  "affinity"?: V1Affinity
  /**
   * List of environment variables to set in the container. Cannot be updated.
   */
  "env"?: Array<V1EnvVar>
  /**
   * List of sources to populate environment variables in the container. The keys defined within a source must be a C_IDENTIFIER. All invalid keys will be reported as an event when the container is starting. When a key exists in multiple sources, the value associated with the last source will take precedence. Values defined by an Env with a duplicate key will take precedence. Cannot be updated.
   */
  "envFrom"?: Array<V1EnvFromSource>
  /**
   * Set host aliases for the Runner Pod
   */
  "hostAliases"?: Array<V1HostAlias>
  /**
   * Runner pod image to use other than default
   */
  "image"?: string
  /**
   * Set up Init Containers for the Runner
   */
  "initContainers"?: Array<V1Container>
  /**
   * Set the NodeSelector for the Runner Pod
   */
  "nodeSelector"?: { [key: string]: string }
  "resource"?: V1ResourceRequirements
  /**
   * Set the Tolerations for the Runner Pod
   */
  "tolerations"?: Array<V1Toleration>
  /**
   * Set Volume Mounts for the Runner Pod
   */
  "volumeMounts"?: Array<V1VolumeMount>
  /**
   * Set Volumes for the Runner Pod
   */
  "volumes"?: Array<V1Volume>

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "affinity",
      baseName: "affinity",
      type: "V1Affinity",
      format: "",
    },
    {
      name: "env",
      baseName: "env",
      type: "Array<V1EnvVar>",
      format: "",
    },
    {
      name: "envFrom",
      baseName: "envFrom",
      type: "Array<V1EnvFromSource>",
      format: "",
    },
    {
      name: "hostAliases",
      baseName: "hostAliases",
      type: "Array<V1HostAlias>",
      format: "",
    },
    {
      name: "image",
      baseName: "image",
      type: "string",
      format: "",
    },
    {
      name: "initContainers",
      baseName: "initContainers",
      type: "Array<V1Container>",
      format: "",
    },
    {
      name: "nodeSelector",
      baseName: "nodeSelector",
      type: "{ [key: string]: string; }",
      format: "",
    },
    {
      name: "resource",
      baseName: "resource",
      type: "V1ResourceRequirements",
      format: "",
    },
    {
      name: "tolerations",
      baseName: "tolerations",
      type: "Array<V1Toleration>",
      format: "",
    },
    {
      name: "volumeMounts",
      baseName: "volumeMounts",
      type: "Array<V1VolumeMount>",
      format: "",
    },
    {
      name: "volumes",
      baseName: "volumes",
      type: "Array<V1Volume>",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerPodTemplateSpec.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerPersistentVolumeClaimTemplate {
  "metadata"?: StorageV1TemplateMetadata
  "spec"?: StorageV1RunnerPersistentVolumeClaimTemplateSpec

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "metadata",
      baseName: "metadata",
      type: "StorageV1TemplateMetadata",
      format: "",
    },
    {
      name: "spec",
      baseName: "spec",
      type: "StorageV1RunnerPersistentVolumeClaimTemplateSpec",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerPersistentVolumeClaimTemplate.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerServiceTemplate {
  "metadata"?: StorageV1TemplateMetadata
  "spec"?: StorageV1RunnerServiceTemplateSpec

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "metadata",
      baseName: "metadata",
      type: "StorageV1TemplateMetadata",
      format: "",
    },
    {
      name: "spec",
      baseName: "spec",
      type: "StorageV1RunnerServiceTemplateSpec",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerServiceTemplate.attributeTypeMap
  }

  public constructor() {}
}

export class StorageV1RunnerServiceTemplateSpec {
  /**
   * type determines how the Service is exposed. Defaults to ClusterIP  Possible enum values:  - `\"ClusterIP\"` means a service will only be accessible inside the cluster, via the cluster IP.  - `\"ExternalName\"` means a service consists of only a reference to an external name that kubedns or equivalent will return as a CNAME record, with no exposing or proxying of any pods involved.  - `\"LoadBalancer\"` means a service will be exposed via an external load balancer (if the cloud provider supports it), in addition to \'NodePort\' type.  - `\"NodePort\"` means a service will be exposed on one port of every node, in addition to \'ClusterIP\' type.
   */
  "type"?: StorageV1RunnerServiceTemplateSpecTypeEnum

  static readonly discriminator: string | undefined = undefined

  static readonly attributeTypeMap: Array<{
    name: string
    baseName: string
    type: string
    format: string
  }> = [
    {
      name: "type",
      baseName: "type",
      type: "StorageV1RunnerServiceTemplateSpecTypeEnum",
      format: "",
    },
  ]

  static getAttributeTypeMap() {
    return StorageV1RunnerServiceTemplateSpec.attributeTypeMap
  }

  public constructor() {}
}

export enum StorageV1RunnerServiceTemplateSpecTypeEnum {
  ClusterIp = "ClusterIP",
  ExternalName = "ExternalName",
  LoadBalancer = "LoadBalancer",
  NodePort = "NodePort",
}
