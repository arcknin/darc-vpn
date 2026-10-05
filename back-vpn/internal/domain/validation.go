package domain

const VPNEndpoint = "vpn.darcanix.com"

func ValidateVPNEndpoint(endpoint string) error {
	if endpoint != VPNEndpoint {
		return NewCodedError(ErrInvalidRequest, "profile endpoint must use the approved VPN endpoint")
	}
	return nil
}

func ValidateRole(role Role) error {
	switch role {
	case RoleFriend, RoleAdmin:
		return nil
	default:
		return NewCodedError(ErrInvalidRequest, "unsupported account role")
	}
}
