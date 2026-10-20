# Verifica que el tag exista: https://quay.io/repository/operator-framework/helm-operator?tab=tags
ARG HELM_OPERATOR_VERSION=v1.39.2
FROM quay.io/operator-framework/helm-operator:${HELM_OPERATOR_VERSION}

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts ${HOME}/helm-charts
WORKDIR ${HOME}
