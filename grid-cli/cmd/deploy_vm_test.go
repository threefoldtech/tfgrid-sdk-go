// Package cmd for parsing command line arguments
package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos_sdk_go/grid-client/zos"
)

func validKeyHex() string  { return strings.Repeat("ab", zos.MyceliumKeyLen) }
func validSeedHex() string { return strings.Repeat("cd", zos.MyceliumIPSeedLen) }

func TestParseMyceliumIdentity(t *testing.T) {
	t.Run("neither supplied means generate", func(t *testing.T) {
		key, seed, err := parseMyceliumIdentity("", "", true)
		require.NoError(t, err)
		assert.Empty(t, key)
		assert.Empty(t, seed)
	})

	t.Run("both supplied are decoded", func(t *testing.T) {
		key, seed, err := parseMyceliumIdentity(validKeyHex(), validSeedHex(), true)
		require.NoError(t, err)
		assert.Len(t, key, zos.MyceliumKeyLen)
		assert.Len(t, seed, zos.MyceliumIPSeedLen)
	})

	// Either alone still changes the address on a redeployment, which defeats the
	// only reason to supply them.
	t.Run("key without seed is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity(validKeyHex(), "", true)
		require.Error(t, err)
	})

	t.Run("seed without key is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity("", validSeedHex(), true)
		require.Error(t, err)
	})

	t.Run("supplying an identity while mycelium is disabled is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity(validKeyHex(), validSeedHex(), false)
		require.Error(t, err)
	})

	t.Run("disabling mycelium without an identity is fine", func(t *testing.T) {
		key, seed, err := parseMyceliumIdentity("", "", false)
		require.NoError(t, err)
		assert.Empty(t, key)
		assert.Empty(t, seed)
	})

	t.Run("a key of the wrong length is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity("aabb", validSeedHex(), true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mycelium key length")
	})

	t.Run("a seed of the wrong length is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity(validKeyHex(), "aabb", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mycelium ip seed length")
	})

	t.Run("a key that is not hex is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity(strings.Repeat("zz", zos.MyceliumKeyLen), validSeedHex(), true)
		require.Error(t, err)
	})

	t.Run("a seed that is not hex is refused", func(t *testing.T) {
		_, _, err := parseMyceliumIdentity(validKeyHex(), strings.Repeat("zz", zos.MyceliumIPSeedLen), true)
		require.Error(t, err)
	})

	// The whole point: the same hex in yields the same bytes out, so a caller that
	// stores the pair can rebuild a machine on the address it had before.
	t.Run("decoding is stable across calls", func(t *testing.T) {
		key1, seed1, err := parseMyceliumIdentity(validKeyHex(), validSeedHex(), true)
		require.NoError(t, err)
		key2, seed2, err := parseMyceliumIdentity(validKeyHex(), validSeedHex(), true)
		require.NoError(t, err)
		assert.Equal(t, key1, key2)
		assert.Equal(t, seed1, seed2)
	})
}
