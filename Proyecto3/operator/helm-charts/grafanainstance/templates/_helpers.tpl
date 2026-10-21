{{- define "grafanainstance.labels" -}}
app.kubernetes.io/name: grafana
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: munk8s-operator
{{- end -}}

{{- define "grafanainstance.selector" -}}
app.kubernetes.io/name: grafana
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
