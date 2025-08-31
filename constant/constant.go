package constant

const (
	VboxCommonPrefix    = "vbox"
	VboxImagePrefix     = VboxCommonPrefix + "-"
	VboxContainerPrefix = VboxCommonPrefix + "-"
	VboxNetwork         = VboxCommonPrefix + "-network"
	VboxNetworkSubnet   = "172.20.0.0/16"
	VboxUser            = "devbox"
)

const (
	DefaultTemplatesDirPath      = "templates"
	DefaultDockerfileName        = "Dockerfile"
	DefaultNetworkDriver         = "bridge"
	DefaultSSHAuthorizedKeysPath = "/home/devbox/.ssh/authorized_keys"
)
