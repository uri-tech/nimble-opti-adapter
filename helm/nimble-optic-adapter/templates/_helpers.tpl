{{/* Generate basic labels */}}
{{- define "nimble-optic-adapter.labels" -}}
helm.sh/chart: {{ include "nimble-optic-adapter.chart" . }}
{{ include "nimble-optic-adapter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels */}}
{{- define "nimble-optic-adapter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nimble-optic-adapter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Generate the name of the service account */}}
{{- define "nimble-optic-adapter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nimble-optic-adapter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end -}}

{{/* Generate the chart name */}}
{{- define "nimble-optic-adapter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/* Generate the name */}}
{{- define "nimble-optic-adapter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/* Generate the fullname */}}
{{- define "nimble-optic-adapter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" (include "nimble-optic-adapter.name" .) .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end -}}