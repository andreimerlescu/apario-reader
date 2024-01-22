package main

import (
	`errors`
	`fmt`
	`strconv`
	`unicode`
)

func ParseDDVersion(version string) (*DDVersion, error) {
	if len(version) < 6 {
		return nil, errors.New("invalid version format. min value is v0.0.1")
	}
	ddv := &DDVersion{}
	octet := 0 // v1.2.3
	octet_value := ""
	bytes := []byte(version)
	for i := 0; i < len(bytes); i++ {
		b := bytes[i]
		r := rune(b)
		if i == 0 {
			continue
		}
		if string(r) == "." {
			if octet == 0 {
				octet = 1
			}
			inner_version, int_err := strconv.Atoi(octet_value)
			if int_err != nil {
				return nil, errors.New("unable to understand octet in version string")
			}
			if octet == 1 {
				ddv.Major = inner_version
				octet_value = ""
				octet += 1
			} else if octet == 2 {
				ddv.Minor = inner_version
				octet_value = ""
				octet += 1
			} else if octet == 3 {
				return nil, errors.New("should not be seeing this error message with the octet == 3 and an inner_version being assigned ddv.Patch")
			}
		} else {
			if unicode.IsDigit(r) {
				octet_value += string(b)
			}
			if i == len(bytes)-1 {
				// last char in the byte slice
				inner_version, int_err := strconv.Atoi(octet_value)
				if int_err != nil {
					return nil, errors.New("unable to understand octet in version string")
				}
				if octet == 1 {
					ddv.Major = inner_version
				} else if octet == 2 {
					ddv.Minor = inner_version
				} else if octet == 3 {
					ddv.Patch = inner_version
					break
				}
			}
		}

	}
	return ddv, nil
}

type DDVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (v *DDVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v *DDVersion) Bump(kind string) {
	if kind == "major" {
		v.Major += 1
	} else if kind == "minor" {
		v.Minor += 1
	} else if kind == "patch" {
		v.Patch += 1
	}
}
