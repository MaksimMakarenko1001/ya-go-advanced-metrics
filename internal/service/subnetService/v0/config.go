package v0

type Config struct {
	TrustedSubnet   string `env:"TRUSTED_SUBNET" json:"trustedSubnet"`
	ValidateEnabled bool   `env:"VALIDATE_ENABLED" envDefault:"true" json:"validateEnabled"`
}
