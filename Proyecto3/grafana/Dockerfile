# Grafana con el plugin Redis datasource preinstalado (sirve para Valkey).
# Asi el pod no depende de internet al arrancar y la imagen sale de Zot.
ARG GRAFANA_VERSION=11.4.0
FROM grafana/grafana:${GRAFANA_VERSION}
RUN grafana cli plugins install redis-datasource
