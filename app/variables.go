// Package app handles all the application settings
package app

import "regexp"

var (
	// DB config
	DB = DBStruct{}

	// ProjectRoot var
	ProjectRoot string

	// Verbose logging
	Verbose bool

	// TempFiles get cleaned up on exit
	TempFiles []string

	// TempDir runtime variable can overridden with flags
	TempDir string

	// OnlyAssets runtime variable set with flags
	OnlyAssets bool

	// OnlyDB runtime variable set with flags
	OnlyDB bool

	// IgnoreResampled runtime variable set with flags
	IgnoreResampled bool

	// ResampledRegex regular expressions should match all common thumbnail manipulations except for
	// resized images as those tend to be linked from HTMLText and aren't auto-generated without a republish
	ResampledRegex = []*regexp.Regexp{
		// Silverstripe 4 and 5
		regexp.MustCompile(`(?i)\_\_(Crop|ExtRewrite|Fill|Fit|Focus|Pad|Quality|Resampled|Scale)([a-z0-9_]*)\.[a-z0-9]{1,4}$`),

		// Silverstripe 3
		regexp.MustCompile(`(?i)\/\_resampled\/(Pad|CMSThumbnail|stripthumbnail|Cropped|Set|Fit|Fill|Scale|Resampled).*\.(jpg|png|jpeg|tiff)`),
	}
)

// DBStruct struct
type DBStruct struct {
	// Host database host
	Host string

	// Username database user
	Username string

	// Password database password
	Password string

	// Name database name
	Name string

	// Port database port (as string)
	Port string

	// Database type (mysql, postgres etc)
	Type string
}
