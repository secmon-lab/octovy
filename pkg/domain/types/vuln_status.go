package types

type VulnStatus string

const (
	VulnStatusActive VulnStatus = "active"
	VulnStatusFixed  VulnStatus = "fixed"
)
