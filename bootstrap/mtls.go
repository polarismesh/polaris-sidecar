package bootstrap

type MTLSConfiguration struct {
	Enable   bool   `yaml:"enable"`
	CAServer string `yaml:"ca_server"`
}
