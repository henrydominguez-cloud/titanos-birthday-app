{{/* Expand the name of the chart. */}}
{{- define "birthday-app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Fully qualified app name. */}}
{{- define "birthday-app.fullname" -}}
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

{{- define "birthday-app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common labels. */}}
{{- define "birthday-app.labels" -}}
helm.sh/chart: {{ include "birthday-app.chart" . }}
{{ include "birthday-app.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels shared by every workload in the release. */}}
{{- define "birthday-app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "birthday-app.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Selector labels for the APP workload only. Adds component=server so that the
app's Service/Deployment/NetworkPolicy/PDB never accidentally select the
in-chart PostgreSQL pods (which share name/instance but use component=postgresql).
*/}}
{{- define "birthday-app.appSelectorLabels" -}}
{{ include "birthday-app.selectorLabels" . }}
app.kubernetes.io/component: server
{{- end -}}

{{/* Service account name. */}}
{{- define "birthday-app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "birthday-app.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* Postgres service name (in-chart database). */}}
{{- define "birthday-app.postgresName" -}}
{{- printf "%s-postgresql" (include "birthday-app.fullname" .) -}}
{{- end -}}

{{/* Effective database host: in-chart postgres unless overridden. */}}
{{- define "birthday-app.dbHost" -}}
{{- if .Values.database.host -}}
{{- .Values.database.host -}}
{{- else -}}
{{- include "birthday-app.postgresName" . -}}
{{- end -}}
{{- end -}}

{{/* Rendered DATABASE_URL (used only when no existingSecret is provided). */}}
{{- define "birthday-app.databaseURL" -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" .Values.database.user .Values.database.password (include "birthday-app.dbHost" .) .Values.database.port .Values.database.name .Values.database.sslmode -}}
{{- end -}}
