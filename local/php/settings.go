package php

import (
	"bytes"

	"github.com/symfony-cli/symfony-cli/envs"
)

type PHPValues []keyVal

type keyVal struct {
	Key string
	Val string
}

func GetPHPINISettings(dir string) *PHPValues {
	values := &PHPValues{}
	values.addBlackfireSocket(dir)
	return values
}

func (v *PHPValues) Bytes() []byte {
	strs := [][]byte{}
	for _, value := range *v {
		strs = append(strs, []byte(value.Key+"="+value.Val))
	}
	return bytes.Join(strs, []byte{'\n'})
}

func (v *PHPValues) merge(m map[string]string) {
	for key, val := range m {
		*v = append(*v, keyVal{Key: key, Val: val})
	}
}

func (v *PHPValues) addBlackfireSocket(dir string) {
	local, err := envs.NewLocal(dir, false)
	if err != nil {
		return
	}
	if local.FindRelationshipPrefix("blackfire", "tcp") == "" {
		return
	}
	if blackfireSocket := envs.AsMap(local)["BLACKFIRE_URL"]; blackfireSocket != "" {
		v.merge(map[string]string{
			"blackfire.agent_socket": blackfireSocket,
		})
	}
}
