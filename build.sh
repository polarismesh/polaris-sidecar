#!/bin/bash

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
folder_name="polaris-sidecar-release_${version}.${GOOS}.${GOARCH}"
pkg_name="${folder_name}.zip"
if [ "${GOOS}" == "windows" ]; then
  bin_name="polaris-sidecar.exe"
fi
echo "GOOS is ${GOOS}, GOARCH is ${GOARCH}, binary name is ${bin_name}"

cd $workdir

# 清理环境
rm -rf ${folder_name}
rm -f "${pkg_name}"

# 编译
rm -f ${bin_name}

# 禁止 CGO_ENABLED 参数打开
export CGO_ENABLED=0

build_date=$(date "+%Y%m%d.%H%M%S")
package="github.com/polarismesh/polaris-sidecar/version"
GOARCH=${GOARCH} GOOS=${GOOS} go build -o ${bin_name} -ldflags="-X ${package}.Version=${version} -X ${package}.BuildDate=${build_date}"

# 设置程序为可执行
chmod +x ${bin_name}

# 打包
mkdir -p ${folder_name}
cp ${bin_name} ${folder_name}
cp polaris-sidecar.yaml ${folder_name}
cp -r tool ${folder_name}/
zip -r "${pkg_name}" ${folder_name}
#md5sum ${pkg_name} > "${pkg_name}.md5sum"

if [[ $(uname -a | grep "Darwin" | wc -l) -eq 1 ]]; then
  md5 ${pkg_name} >"${pkg_name}.md5sum"
else
  md5sum ${pkg_name} >"${pkg_name}.md5sum"
fi
