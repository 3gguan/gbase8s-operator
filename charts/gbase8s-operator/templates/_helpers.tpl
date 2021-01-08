{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "gbase8s-operator.name" -}}
{{- default (printf "%s-controller-manager" .Chart.Name) .Values.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gbase8s-operator.roleName" -}}
{{- default (printf "%s-manager-role" .Chart.Name) .Values.roleName | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gbase8s-operator.roleBindingName" -}}
{{- default (printf "%s-manager-rolebinding" .Chart.Name) .Values.roleBindingName | trunc 63 | trimSuffix "-" -}}
{{- end -}}