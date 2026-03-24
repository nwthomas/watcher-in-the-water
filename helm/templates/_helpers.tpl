{{/*
Expand the name of the chart.
*/}}
{{- define "watcher-in-the-water.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "watcher-in-the-water.fullname" -}}
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
{{- define "watcher-in-the-water.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "watcher-in-the-water.labels" -}}
helm.sh/chart: {{ include "watcher-in-the-water.chart" . }}
{{ include "watcher-in-the-water.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "watcher-in-the-water.selectorLabels" -}}
app.kubernetes.io/name: {{ include "watcher-in-the-water.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "watcher-in-the-water.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "watcher-in-the-water.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "watcher-in-the-water.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Create configmap name
*/}}
{{- define "watcher-in-the-water.configmapName" -}}
{{- printf "%s-config" (include "watcher-in-the-water.fullname" .) }}
{{- end }}

{{/*
Create secret name
*/}}
{{- define "watcher-in-the-water.secretName" -}}
{{- printf "%s-secret" (include "watcher-in-the-water.fullname" .) }}
{{- end }}
