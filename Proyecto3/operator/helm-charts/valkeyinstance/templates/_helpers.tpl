{{- define "valkeyinstance.labels" -}}
app.kubernetes.io/name: valkey
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: munk8s-operator
{{- end -}}

{{- define "valkeyinstance.selector" -}}
app.kubernetes.io/name: valkey
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
