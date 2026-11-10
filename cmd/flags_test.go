package cmd

import (
	"testing"

	"github.com/axllent/ssbak/app"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOnlyFlags(t *testing.T) {
	tests := []struct {
		args       []string
		onlyDB     bool
		onlyAssets bool
		wantErr    string
	}{
		{args: []string{}},
		{args: []string{"--db"}, onlyDB: true},
		{args: []string{"--db=true"}, onlyDB: true},
		{args: []string{"--assets"}, onlyAssets: true},
		{args: []string{"--db=false"}, onlyAssets: true},
		{args: []string{"--assets=false"}, onlyDB: true},
		{args: []string{"--db=false", "--assets"}, onlyAssets: true},
		{args: []string{"--db", "--assets=false"}, onlyDB: true},
		{args: []string{"--db", "--assets"}, wantErr: "cannot use --assets and --db"},
		{args: []string{"--db=false", "--assets=false"}, wantErr: "nothing to do"},
	}

	defer func() {
		app.OnlyDB = false
		app.OnlyAssets = false
	}()

	for _, tt := range tests {
		app.OnlyDB = false
		app.OnlyAssets = false

		cmd := &cobra.Command{}
		cmd.Flags().BoolVar(&app.OnlyDB, "db", false, "")
		cmd.Flags().BoolVar(&app.OnlyAssets, "assets", false, "")
		require.NoError(t, cmd.ParseFlags(tt.args), "args: %v", tt.args)

		err := resolveOnlyFlags(cmd)
		if tt.wantErr != "" {
			require.Error(t, err, "args: %v", tt.args)
			assert.Contains(t, err.Error(), tt.wantErr, "args: %v", tt.args)
			continue
		}

		require.NoError(t, err, "args: %v", tt.args)
		assert.Equal(t, tt.onlyDB, app.OnlyDB, "OnlyDB, args: %v", tt.args)
		assert.Equal(t, tt.onlyAssets, app.OnlyAssets, "OnlyAssets, args: %v", tt.args)
	}
}
