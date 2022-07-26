package main

import "strings"

func intersection(s1, s2 []string) (res []string) {
	hash := make(map[string]struct{})

	for _, e := range s1 {
		hash[e] = struct{}{}
	}
	for _, e := range s2 {
		if _, ok := hash[e]; ok {
			res = append(res, e)
		}
	}
	return
}

func rpOSMIdConv(osmId string) string {
	return strings.Replace(osmId, "-", "R", 1)
}
