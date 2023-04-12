// Package deployer for project deployment
package deployer

import "math/rand"

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randName(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
