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

	// The key may arrive named rather than written. The seed is deliberately not
	// given the same treatment: it selects an address within the network the key
	// defines, so knowing it grants nothing.
	t.Run("a named environment variable is read", func(t *testing.T) {
		t.Setenv("TEST_MYCELIUM_KEY", validKeyHex())
		resolved, err := resolveMyceliumKeyHex("", "TEST_MYCELIUM_KEY")
		require.NoError(t, err)
		assert.Equal(t, validKeyHex(), resolved)
	})

	t.Run("naming no variable passes the value through", func(t *testing.T) {
		resolved, err := resolveMyceliumKeyHex(validKeyHex(), "")
		require.NoError(t, err)
		assert.Equal(t, validKeyHex(), resolved)
	})

	t.Run("naming neither still means generate", func(t *testing.T) {
		resolved, err := resolveMyceliumKeyHex("", "")
		require.NoError(t, err)
		assert.Empty(t, resolved)
	})

	t.Run("both a value and a variable is refused", func(t *testing.T) {
		_, err := resolveMyceliumKeyHex(validKeyHex(), "TEST_MYCELIUM_KEY")
		require.Error(t, err)
	})

	// An unset variable is a caller mistake worth naming precisely, because the
	// alternative — falling back to generating one — hands back a machine on an
	// address the caller did not choose and believes it did.
	t.Run("an unset variable is refused and named", func(t *testing.T) {
		_, err := resolveMyceliumKeyHex("", "TEST_MYCELIUM_KEY_UNSET")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TEST_MYCELIUM_KEY_UNSET")
	})

	t.Run("an empty variable is refused and named", func(t *testing.T) {
		t.Setenv("TEST_MYCELIUM_KEY_EMPTY", "")
		_, err := resolveMyceliumKeyHex("", "TEST_MYCELIUM_KEY_EMPTY")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TEST_MYCELIUM_KEY_EMPTY")
	})

	// The reason the flag exists at all: what went wrong is reported, what it held
	// is not.
	t.Run("a rejected variable never echoes what it held", func(t *testing.T) {
		t.Setenv("TEST_MYCELIUM_KEY_SECRET", "")
		_, err := resolveMyceliumKeyHex("", "TEST_MYCELIUM_KEY_SECRET")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), validKeyHex())

		t.Setenv("TEST_MYCELIUM_KEY_BAD", validKeyHex())
		resolved, err := resolveMyceliumKeyHex("", "TEST_MYCELIUM_KEY_BAD")
		require.NoError(t, err)
		_, _, err = parseMyceliumIdentity(resolved, "aabb", true)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), validKeyHex())
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
