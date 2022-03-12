# /bin/bash

# 先构建部署包
bash build.sh

# 构建安装包

package_name="polaris-sidecar-install.zip"
folder_name="polaris-sidecar-install"

mkdir -p ${folder_name}

deploy_package=$(ls -lstrh | awk '{print $10}' | grep -E "^polaris-sidecar-release.+zip$")

mv ${deploy_package} ${folder_name}
cp ./deploy/vm/*.sh ${folder_name}

zip -r "${package_name}" ${folder_name}
rm -rf ${folder_name}
rm -rf polaris-sidecar-release_*
