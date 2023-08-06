{{/* Generate basic labels */}}
{{- define "nimble-opti-adapter.labels" -}}
helm.sh/chart: {{ include "nimble-opti-adapter.chart" . }}
{{ include "nimble-opti-adapter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels */}}
{{- define "nimble-opti-adapter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nimble-opti-adapter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Generate the name of the service account */}}
{{- define "nimble-opti-adapter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nimble-opti-adapter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end -}}

{{/* Generate the chart name */}}
{{- define "nimble-opti-adapter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/* Generate the name */}}
{{- define "nimble-opti-adapter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/* Generate the fullname */}}
{{- define "nimble-opti-adapter.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s" (include "nimble-opti-adapter.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end -}}