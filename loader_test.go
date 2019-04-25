// nolint: govet
package konghcl

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"
)

const testConfig = `
	flag-name = "hello world"
	int-flag = 10
	float-flag = 10.5
	slice-flag = [1, 2, 3]

	prefix {
		prefixed-flag = "prefixed flag"
	}
	group {
		grouped-flag = "grouped flag"
		embedded-flag = "embedded flag"
	}
	map-flag = {
		key = "value"
	}
	// Multiple keys are merged.
	mapped = {
		left = "left"
	}
	mapped = {
		right = "right"
	}
`

type mapperValue struct {
	Left  string
	Right string
}

func (m *mapperValue) Decode(ctx *kong.DecodeContext) error {
	return DecodeValue(ctx, m)
}

func TestHCL(t *testing.T) {
	type Embedded struct {
		EmbeddedFlag string
	}
	type CLI struct {
		FlagName     string
		IntFlag      int
		FloatFlag    float64
		SliceFlag    []int
		GroupedFlag  string `group:"group"`
		PrefixedFlag string `prefix:"prefix-"`
		Embedded     `group:"group"`
		MapFlag      map[string]string
		Mapped       mapperValue
	}

	t.Run("FromResolver", func(t *testing.T) {
		var cli CLI
		r := strings.NewReader(testConfig)
		resolver, err := Loader(r)
		require.NoError(t, err)
		parser, err := kong.New(&cli, kong.Resolvers(resolver))
		require.NoError(t, err)
		_, err = parser.Parse(nil)
		require.NoError(t, err)
		require.Equal(t, "hello world", cli.FlagName)
		require.Equal(t, "grouped flag", cli.GroupedFlag)
		require.Equal(t, "prefixed flag", cli.PrefixedFlag)
		require.Equal(t, "embedded flag", cli.EmbeddedFlag)
		require.Equal(t, 10, cli.IntFlag)
		require.Equal(t, 10.5, cli.FloatFlag)
		require.Equal(t, []int{1, 2, 3}, cli.SliceFlag)
		require.Equal(t, map[string]string{"key": "value"}, cli.MapFlag)
		require.Equal(t, mapperValue{Left: "left", Right: "right"}, cli.Mapped)
	})

	t.Run("FragmentFromFlag", func(t *testing.T) {
		var cli CLI
		parser, err := kong.New(&cli)
		require.NoError(t, err)
		_, err = parser.Parse([]string{"--mapped", `
		left = "LEFT"
		right = "RIGHT"
		`})
		require.NoError(t, err)
		require.Equal(t, mapperValue{Left: "LEFT", Right: "RIGHT"}, cli.Mapped)
	})

	t.Run("FragmentFromFile", func(t *testing.T) {
		w, err := ioutil.TempFile("", "kong-hcl-")
		require.NoError(t, err)
		_, err = w.Write([]byte(`
		left = "LEFT"
		right = "RIGHT"
		`))
		require.NoError(t, err)
		_ = w.Close()
		defer os.Remove(w.Name())

		var cli CLI
		parser, err := kong.New(&cli)
		require.NoError(t, err)
		_, err = parser.Parse([]string{"--mapped", w.Name()})
		require.NoError(t, err)
		require.Equal(t, mapperValue{Left: "LEFT", Right: "RIGHT"}, cli.Mapped)
	})
}

func TestHCLValidation(t *testing.T) {
	type command struct {
		CommandFlag string
	}
	var cli struct {
		Command command `cmd:""`
		Flag    string
	}
	resolver, err := Loader(strings.NewReader(`
		invalid-flag = true
	`))
	require.NoError(t, err)
	parser, err := kong.New(&cli, kong.Resolvers(resolver))
	require.NoError(t, err)
	_, err = parser.Parse([]string{"command"})
	require.EqualError(t, err, "unknown configuration key \"invalid-flag\"")
}
