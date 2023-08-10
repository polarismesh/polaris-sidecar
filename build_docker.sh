#!/bin/bash

set -e

DOCKER_IMAGE_RPO="${DOCKER_RPO}"
if [[ "${DOCKER_RPO}" == "" ]]; then
  DOCKER_IMAGE_RPO="polarismesh"
fi

if [ $# != 1 ]; then
  echo "e.g.: bash $0 v1.0"
  exit 1
fi

docker_tag=$1

echo "docker repository : ${DOCKER_IMAGE_RPO}/polaris-sidecar, tag : ${docker_tag}"

bash build.sh ${docker_tag}

if [ $? != 0 ]; then
  echo "build polaris-sidecar failed"
  exit 1
fi

docker build --network=host -t ${DOCKER_IMAGE_RPO}/polaris-sidecar:${docker_tag} ./
docker push ${DOCKER_IMAGE_RPO}/polaris-sidecar:${docker_tag}

pre_release=$(echo ${docker_tag} | egrep "(alpha|beta|rc)" | wc -l)
if [ ${pre_release} == 0 ]; then
  docker tag ${DOCKER_IMAGE_RPO}/polaris-sidecar:${docker_tag} ${DOCKER_IMAGE_RPO}/polaris-sidecar:latest
  docker push ${DOCKER_IMAGE_RPO}/polaris-sidecar:latest
fi
