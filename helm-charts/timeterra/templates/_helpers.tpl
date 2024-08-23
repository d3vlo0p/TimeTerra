{{/*
Expand the name of the chart.
*/}}
{{- define "timeterra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "timeterra.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "timeterra.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "timeterra.labels" -}}
helm.sh/chart: {{ include "timeterra.chart" . }}
{{ include "timeterra.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "timeterra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "timeterra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "timeterra.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "timeterra.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
define namespaces to watch
*/}}
{{- define "timeterra.settings.watchNamespaces" -}}
{{- if .Values.settings -}}
    {{- if not .Values.settings.watchAllNamespaces -}}
- name: WATCH_NAMESPACE
  value: {{ default .Release.Namespace .Values.settings.watchNamespaces  }}
    {{- end }}
{{- else -}}
- name: WATCH_NAMESPACE
  value: {{ .Release.Namespace }}
{{- end }}
{{- end }}