{{/*
Expand the name of the chart.
*/}}
{{- define "golang-server-boilerplate.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "golang-server-boilerplate.fullname" -}}
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
{{- define "golang-server-boilerplate.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "golang-server-boilerplate.labels" -}}
helm.sh/chart: {{ include "golang-server-boilerplate.chart" . }}
{{ include "golang-server-boilerplate.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "golang-server-boilerplate.selectorLabels" -}}
app.kubernetes.io/name: {{ include "golang-server-boilerplate.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "golang-server-boilerplate.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "golang-server-boilerplate.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "golang-server-boilerplate.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Create configmap name
*/}}
{{- define "golang-server-boilerplate.configmapName" -}}
{{- printf "%s-config" (include "golang-server-boilerplate.fullname" .) }}
{{- end }}

{{/*
Create secret name
*/}}
{{- define "golang-server-boilerplate.secretName" -}}
{{- printf "%s-secret" (include "golang-server-boilerplate.fullname" .) }}
{{- end }}
