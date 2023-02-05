package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jrudio/go-plex-client"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

// main
func main() {
	app := cli.App{
		Name:        "plax",
		Usage:       "Plex Local Asset eXporter",
		Description: "Export posters and background assets from Plex",
		Version:     "0.1.0",
		Authors: []*cli.Author{
			{
				Name:  "Jeff Bailey",
				Email: "jeff@blahspam.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Plex base URL",
				Value:    "http://127.0.0.1:32400",
				EnvVars:  []string{"PLEX_URL"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "Plex authentication token",
				EnvVars:  []string{"PLEX_TOKEN"},
				Required: true,
			},
			&cli.StringFlag{
				Name:  "output-dir",
				Usage: "alternate directory for exported assets",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "less verbose output",
			},
		},
		Action: func(ctx *cli.Context) error {
			p, err := plex.New(strings.TrimSuffix(ctx.String("url"), "/"), ctx.String("token"))
			if err != nil {
				return cli.Exit(err, 1)
			}

			libs, err := p.GetLibraries()
			if err != nil {
				return cli.Exit(err, 2)
			}

			outputDir := ctx.String("output-dir")

			for i := range libs.MediaContainer.Directory {
				dir := &libs.MediaContainer.Directory[i]

				contents, err := p.GetLibraryContent(dir.Key, "")
				if err != nil {
					logError("error retrieving library contents: %s", err)
					continue
				}

				var bar *progressbar.ProgressBar
				if !ctx.Bool("quiet") {
					bar = progressbar.NewOptions(len(contents.MediaContainer.Metadata),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionSetDescription(fmt.Sprintf("[%d/%d] Exporting %s", i+1, len(libs.MediaContainer.Directory), dir.Title)),
						progressbar.OptionShowCount(),
						progressbar.OptionSetTheme(progressbar.Theme{
							Saucer:        "[green]=[reset]",
							SaucerHead:    "[green]>[reset]",
							SaucerPadding: " ",
							BarStart:      "[",
							BarEnd:        "]",
						}))
				}

				for _, meta := range contents.MediaContainer.Metadata {
					switch meta.Type {
					case "movie":
						exportMovieAssets(p, meta, outputDir)
					case "show":
						exportShowAssets(p, meta, outputDir)
					}

					if bar != nil {
						_ = bar.Add(1)
					}
				}
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		logError(err.Error())
	}
}

// download an asset from plex and store it at the location specified.
func download(p *plex.Plex, assetPath string, file string) error {
	if assetPath == "" {
		return nil
	}

	req, err := http.NewRequest(http.MethodGet, p.URL+assetPath, nil)
	req.Header.Add("X-Plex-Token", p.Token)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error retrieving art: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return fmt.Errorf("error creating dir: %w", err)
	}

	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer f.Close()

	_, _ = io.Copy(f, resp.Body)
	return nil
}

// getAssets returns the slice of assets for a content item.
func exportAssets(p *plex.Plex, meta plex.Metadata, outputDir string) {
	if err := download(p, meta.Thumb, outputDir+"/poster.jpg"); err != nil {
		logError("error exporting poster for %s: %s", meta.Title, err)
	}
	if err := download(p, meta.Art, outputDir+"/background.jpg"); err != nil {
		logError("error storing background for %s: %s", meta.Title, err)
	}
}

// export movie assets
func exportMovieAssets(p *plex.Plex, movieMeta plex.Metadata, outputDir string) {
	movieDir := filepath.Join(outputDir, getContentDir(p, movieMeta))
	exportAssets(p, movieMeta, movieDir)
}

// export show and season assets
func exportShowAssets(p *plex.Plex, showMeta plex.Metadata, outputDir string) {
	// show assets
	showDir := filepath.Join(outputDir, getContentDir(p, showMeta))
	exportAssets(p, showMeta, showDir)

	// season assets
	seasons, err := p.GetEpisodes(showMeta.RatingKey)
	if err != nil {
		logError("error retrieving %s seasons: %s", showMeta.Title)
		return
	}

	for _, seasonMeta := range seasons.MediaContainer.Metadata {
		seasonDir := filepath.Join(outputDir, getContentDir(p, seasonMeta))
		exportAssets(p, seasonMeta, seasonDir)
	}
}

// getContentDir returns the directory where a content's assets are to be
// stored.
func getContentDir(p *plex.Plex, meta plex.Metadata) string {
	switch meta.Type {
	case "movie", "episode":
		for _, m := range meta.Media {
			for _, p := range m.Part {
				if p.File != "" {
					return filepath.Dir(p.File)
				}
			}
		}
	case "season":
		episodes, err := p.GetEpisodes(meta.RatingKey)
		if err != nil {
			logError("error retrieving episodes: %s", err)
			return ""
		}
		for _, episodeMeta := range episodes.MediaContainer.Metadata {
			// season directory is identical to episode directory
			p := getContentDir(p, episodeMeta)
			if p != "" {
				return p
			}
		}
	case "show":
		seasons, err := p.GetEpisodes(meta.RatingKey)
		if err != nil {
			logError("error retrieving seasons: %s", err)
			return ""
		}
		for _, seasonMeta := range seasons.MediaContainer.Metadata {
			// show directory is parent of season directory
			p := getContentDir(p, seasonMeta)
			if p != "" {
				return strings.TrimSuffix(p, filepath.Base(p))
			}
		}
	default:
		logError("unsupported metadata type: %s", meta.Type)
	}
	return ""
}

// logError for logging errors.
func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args)
}
