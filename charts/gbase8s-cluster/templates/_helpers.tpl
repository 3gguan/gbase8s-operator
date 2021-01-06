{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "gbase8s-cluster.configmap" -}}
{{- default (printf "%s-conf" .Values.name) .Values.configmap | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gbase8s-cluster.secret" -}}
{{- default (printf "%s-secret" .Values.name) .Values.secret | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gbase8s-cluster.cm.svc" -}}
{{- default (printf "%s-service" .Values.name) .Values.cmService | trunc 63 | trimSuffix "-" -}}
{{- end -}}