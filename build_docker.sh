#!/bin/bash

if [ $# != 3 ]; then
    echo "e.g.: bash $0 v1.0 docker_username docekr_user_password"
    exit 1
fi

docker_tag=$1

echo "docker repository : polarismesh/polaris-sidecar, tag : ${docker_tag}"

bash build.sh

if [ $? != 0 ]; then
    echo "build polaris-sidecar failed"
    exit 1
fi

docker build --network=host -t polarismesh/polaris-sidecar:${docker_tag} ./

docker push polarismesh/polaris-sidecar:${docker_tag}
docker tag polarismesh/polaris-sidecar:${docker_tag} polarismesh/polaris-sidecar:latest
docker push polarismesh/polaris-sidecar:latest
