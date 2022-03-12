#! /bin/bash

dns_back_dir="./"
while getopts "back_dir" arg; do #选项后面的冒号表示该选项需要参数
	case ${arg} in
	back_dir)
		dns_back_dir="${arg}"
		;;
	?) #当有不认识的选项的时候arg为?
		echo "unkonw argument"
		exit 1
		;;
	esac
done

echo "[INFO] input param: dns_backdir = ${dns_back_dir}"

function uninstall_polaris_sidecar() {
	echo -e "[INFO] start to stop polaris-sidecar process"
	local ret=$(ps -ef | grep polaris-sidecar | awk '{print $2}' | xargs kill -15)
	echo -e "[INFO] kill polaris-sidecar ret=${ret}"
}

function rollback_dns_conf() {
	echo "[INFO] start rollback /etc/resolv.conf..."
	if [ ! -d "${dns_back_dir}" ]; then
		echo "[ERROR] resolv.conf back_dir not exist"
		exit 1
	fi

	# 找到最新的一个 resolv.conf 备份文件
	local last_back_file=$(ls -lstrh "${dns_back_dir}" | grep resolv.conf.bak_ | sort | tail -1 | awk '{print $NF}')

	echo "" >/etc/resolv.conf
	cat "${dns_back_dir}/${last_back_file}" | while read line; do
		echo "[INFO] ${line}"
		echo "${line}" >>/etc/resolv.conf
	done
}

# 停止 polaris-sidecar 进程
uninstall_polaris_sidecar
# 恢复 /etc/resolv.conf 配置文件
rollback_dns_conf
