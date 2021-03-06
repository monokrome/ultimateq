package data

// UserModes provides basic modes for channels and users.
type UserModes struct {
	modes byte
	*UserModeKinds
}

// NewUserModes creates a new usermodes using the metadata instance for
// reference information.
func NewUserModes(u *UserModeKinds) *UserModes {
	return &UserModes{
		UserModeKinds: u,
	}
}

// SetMode sets the mode given.
func (u *UserModes) SetMode(mode rune) {
	u.modes |= u.GetModeBit(mode)
}

// HasMode checks if the user has the given mode.
func (u *UserModes) HasMode(mode rune) bool {
	bit := u.GetModeBit(mode)
	return bit != 0 && (bit == u.modes&bit)
}

// UnsetMode unsets the mode given.
func (u *UserModes) UnsetMode(mode rune) {
	u.modes &= ^u.GetModeBit(mode)
}

// String turns user modes into a string.
func (u *UserModes) String() string {
	ret := ""
	for i := 0; i < len(u.modeInfo); i++ {
		if u.HasMode(u.modeInfo[i][0]) {
			ret += string(u.modeInfo[i][0])
		}
	}
	return ret
}

// StringSymbols turns user modes into a string but uses mode chars instead.
func (u *UserModes) StringSymbols() string {
	ret := ""
	for i := 0; i < len(u.modeInfo); i++ {
		if u.HasMode(u.modeInfo[i][0]) {
			ret += string(u.modeInfo[i][1])
		}
	}
	return ret
}
