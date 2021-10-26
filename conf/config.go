package conf

type DnsConfig struct {
	SidecarAddr            string `json:"SidecarAddr"`
	SidecarAdminPort       string `json:"SidecarAdminPort"`
	ReloadDnsCacheInterval int    `yaml:"ReloadDnsCacheInterval"`
	DNSAnswerIp            string `yaml:"DNSAnswerIp"`
}

func (d *DnsConfig) SetDefault() {
	if d.SidecarAddr == "" {
		d.SidecarAddr = "127.0.0.1"
	}
	if d.SidecarAdminPort == "" {
		d.SidecarAdminPort = "15000"
	}
	if d.ReloadDnsCacheInterval == 0 {
		d.ReloadDnsCacheInterval = 2
	}
	if d.DNSAnswerIp == "" {
		d.DNSAnswerIp = "10.4.4.4"
	}
}
