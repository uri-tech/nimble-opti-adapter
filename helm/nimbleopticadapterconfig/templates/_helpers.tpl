{{/* Generate basic labels */}}
{{- define "nimbleopticadapterconfig.labels" -}}
helm.sh/chart: {{ include "nimbleopticadapterconfig.chart" . }}
{{ include "nimbleopticadapterconfig.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels */}}
{{- define "nimbleopticadapterconfig.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nimbleopticadapterconfig.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Generate the name of the service account */}}
{{- define "nimbleopticadapterconfig.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nimbleopticadapterconfig.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end -}}