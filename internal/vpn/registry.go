package vpn

import (
	"golang.org/x/sys/windows/registry"
)

// regOpenKey / regEnumKey / regGetString — thin-обгортки над registry API.
const regHKLM = "HKEY_LOCAL_MACHINE"

const regKeyRead = registry.READ | registry.WOW64_64KEY

func regOpenKey(root string, path string, access uint32) (registry.Key, error) {
	var key registry.Key
	switch root {
	case regHKLM:
		key = registry.LOCAL_MACHINE
	default:
		key = registry.CURRENT_USER
	}
	return registry.OpenKey(key, path, access)
}

func regGetString(k registry.Key, name string) (string, error) {
	val, _, err := k.GetStringValue(name)
	return val, err
}
