package caclient

import "io/ioutil"

const SATLocation = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func ServiceAccountToken() string {
	token, _ := ioutil.ReadFile(SATLocation)
	return string(token)
}
