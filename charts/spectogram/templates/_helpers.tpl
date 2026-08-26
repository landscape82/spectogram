{{/*
Expand the name of the chart.
*/}}
{{- define "spectogram.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "spectogram.fullname" -}}
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

{{- define "spectogram.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "spectogram.labels" -}}
helm.sh/chart: {{ include "spectogram.chart" . }}
{{ include "spectogram.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "spectogram.selectorLabels" -}}
app.kubernetes.io/name: {{ include "spectogram.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "spectogram.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "spectogram.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "spectogram.pvcName" -}}
{{- if .Values.persistence.existingClaim }}
{{- .Values.persistence.existingClaim }}
{{- else }}
{{- include "spectogram.fullname" . }}-data
{{- end }}
{{- end }}
