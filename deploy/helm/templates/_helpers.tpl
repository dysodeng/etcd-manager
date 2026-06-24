{{/*
Expand the name of the chart.
*/}}
{{- define "etcd-manager.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this
(by the DNS naming spec).
*/}}
{{- define "etcd-manager.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Chart name and version label.
*/}}
{{- define "etcd-manager.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "etcd-manager.labels" -}}
helm.sh/chart: {{ include "etcd-manager.chart" . }}
{{ include "etcd-manager.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels (scoped to .ctx which can be the root context or a sub map
carrying Chart/Release/Templates fields). Used by all workloads.
*/}}
{{- define "etcd-manager.selectorLabels" -}}
app.kubernetes.io/name: {{ include "etcd-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Backend-specific selector labels.
*/}}
{{- define "etcd-manager.backendSelectorLabels" -}}
app.kubernetes.io/name: {{ include "etcd-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: backend
{{- end -}}

{{/*
Frontend-specific selector labels.
*/}}
{{- define "etcd-manager.frontendSelectorLabels" -}}
app.kubernetes.io/name: {{ include "etcd-manager.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: frontend
{{- end -}}

{{/*
Fully qualified backend name.
*/}}
{{- define "etcd-manager.backend.fullname" -}}
{{- printf "%s-backend" (include "etcd-manager.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified frontend name.
*/}}
{{- define "etcd-manager.frontend.fullname" -}}
{{- printf "%s-frontend" (include "etcd-manager.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified etcd cluster name (bundled).
*/}}
{{- define "etcd-manager.etcd.fullname" -}}
{{- printf "%s-etcd" (include "etcd-manager.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Resolve the effective image tag: explicit value first, then Chart appVersion,
finally the Chart version.
*/}}
{{- define "etcd-manager.imageTag" -}}
{{- $tag := .tag -}}
{{- if not $tag -}}{{- $tag = .ctx.Chart.AppVersion -}}{{- end -}}
{{- if not $tag -}}{{- $tag = .ctx.Chart.Version -}}{{- end -}}
{{- $tag -}}
{{- end -}}

{{/*
Build a fully-qualified image reference.
Params: ctx = $ (root), image = .Values.<svc>.image
Output:  [<registry>/]<repository>:<tag>
*/}}
{{- define "etcd-manager.image" -}}
{{- $registry := .ctx.Values.global.imageRegistry -}}
{{- $repo := required "image.repository is required" .image.repository -}}
{{- $tag := include "etcd-manager.imageTag" (dict "ctx" .ctx "tag" .image.tag) -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry $repo $tag -}}
{{- else -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end -}}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "etcd-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "etcd-manager.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Render standard Kubernetes metadata block (name + labels + annotations).
Usage: include "etcd-manager.metadata" (dict "ctx" . "name" "foo" "annotations" $anno)
*/}}
{{- define "etcd-manager.metadata" -}}
name: {{ .name | quote }}
labels:
{{- include "etcd-manager.labels" .ctx | nindent 2 }}
{{- with .labels }}
{{- toYaml . | nindent 2 }}
{{- end }}
{{- with .annotations }}
annotations:
{{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}

{{/*
Build the etcd --initial-cluster string for a StatefulSet of N replicas.
Produces: "name-0=http://name-0.<svc>.ns.svc:2380,name-1=...,..."
*/}}
{{- define "etcd-manager.etcd.initialCluster" -}}
{{- $root := . -}}
{{- $svc := include "etcd-manager.etcd.fullname" . -}}
{{- $peerPort := .Values.etcd.service.peerPort -}}
{{- range $i, $e := until (int .Values.etcd.replicaCount) -}}
{{- if $i }},{{ end -}}
{{ $root.Release.Name }}-etcd-{{ $i }}=http://{{ $root.Release.Name }}-etcd-{{ $i }}.{{ $svc }}.{{ $root.Release.Namespace }}.svc:{{ $peerPort }}
{{- end -}}
{{- end -}}
