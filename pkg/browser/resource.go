package browser

import (
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// ResourceType identifies a category of browser resource.
type ResourceType int

const (
	// ResourceImage blocks image requests (JPEG, PNG, GIF, WebP, SVG…).
	ResourceImage ResourceType = iota
	// ResourceStylesheet blocks CSS files.
	ResourceStylesheet
	// ResourceFont blocks web font files (WOFF, WOFF2, TTF, EOT).
	ResourceFont
	// ResourceMedia blocks audio and video resources.
	ResourceMedia
)

// resourcePatterns maps each ResourceType to a set of URL glob patterns
// understood by Chrome's network.SetBlockedURLs API.
var resourcePatterns = map[ResourceType][]string{
	ResourceImage: {
		"*.jpg", "*.jpeg", "*.png", "*.gif", "*.webp", "*.svg",
		"*.ico", "*.bmp", "*.tiff",
	},
	ResourceStylesheet: {
		"*.css",
	},
	ResourceFont: {
		"*.woff", "*.woff2", "*.ttf", "*.otf", "*.eot",
	},
	ResourceMedia: {
		"*.mp4", "*.webm", "*.ogg", "*.mp3", "*.wav", "*.avi",
		"*.mov", "*.m4v", "*.m4a",
	},
}

// ResourceFilter holds the resource blocking configuration.
// Use BlockAll to block all non-essential resources, or set individual
// flags for finer control.
type ResourceFilter struct {
	// BlockImages, if true, blocks all image requests.
	BlockImages bool
	// BlockStylesheets, if true, blocks CSS files.
	BlockStylesheets bool
	// BlockFonts, if true, blocks font files.
	BlockFonts bool
	// BlockMedia, if true, blocks audio/video files.
	BlockMedia bool
	// ExtraPatterns holds additional URL glob patterns to block.
	ExtraPatterns []string
}

// BlockAll returns a ResourceFilter that blocks images, stylesheets, fonts,
// and media — suitable for text-only crawls.
func BlockAll() ResourceFilter {
	return ResourceFilter{
		BlockImages:      true,
		BlockStylesheets: true,
		BlockFonts:       true,
		BlockMedia:       true,
	}
}

// patterns returns the combined set of URL globs to pass to Chrome.
func (f ResourceFilter) patterns() []string {
	var out []string
	if f.BlockImages {
		out = append(out, resourcePatterns[ResourceImage]...)
	}
	if f.BlockStylesheets {
		out = append(out, resourcePatterns[ResourceStylesheet]...)
	}
	if f.BlockFonts {
		out = append(out, resourcePatterns[ResourceFont]...)
	}
	if f.BlockMedia {
		out = append(out, resourcePatterns[ResourceMedia]...)
	}
	out = append(out, f.ExtraPatterns...)
	return out
}

// apply returns chromedp actions that install the block list into the tab.
// Returns nil when no patterns are configured.
func (f ResourceFilter) apply() []chromedp.Action {
	patterns := f.patterns()
	if len(patterns) == 0 {
		return nil
	}
	blocked := make([]*network.BlockPattern, len(patterns))
	for i, p := range patterns {
		blocked[i] = &network.BlockPattern{URLPattern: p, Block: true}
	}
	return []chromedp.Action{
		network.Enable(),
		network.SetBlockedURLs().WithURLPatterns(blocked),
	}
}
