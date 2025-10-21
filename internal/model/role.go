package model

type UserRole int8

const (
	UserRoleDefault = UserRole(iota)
	UserRoleAdmin
	UserRolePremium
)

func ParseUserRole(s string) UserRole {
	switch s {
	case "admin":
		return UserRoleAdmin
	case "premium":
		return UserRolePremium
	default:
		return UserRoleDefault
	}
}
