# /bin/bash

version=$1

# 先构建部署包
bash build.sh

# 构建安装包

set -e

if [ $# -gt 0 ]; then
  version="$1"
else
  current=`date "+%Y-%m-%d %H:%M:%S"`
  timeStamp=`date -d "$current" +%s`
  currentTimeStamp=$(((timeStamp*1000+10#`date "+%N"`/1000000)/1000))
  version="$currentTimeStamp"
fi
workdir=$(dirname $(realpath $0))
bin_name="polaris-sidecar"
if [ "${GOOS}" == "" ]; then
  GOOS=$(go env GOOS)
fi
if [ "${GOARCH}" == "" ]; then
  GOARCH=$(go env GOARCH)
fi

package_name="polaris-sidecar-install_${version}.${GOOS}.${GOARCH}.zip"
folder_name="polaris-sidecar-install"

mkdir -p ${folder_name}

deploy_package=$(ls -lstrh | awk '{print $10}' | grep -E "^polaris-sidecar-release.+zip$")

mv ${deploy_package} ${folder_name}
cp ./deploy/vm/*.sh ${folder_name}

zip -r "${package_name}" ${folder_name}
rm -rf ${folder_name}
rm -rf polaris-sidecar-release_*
