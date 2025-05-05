// Package main is the entrypoint for the plax application.
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/blahspam/plax/plex"
	"github.com/urfave/cli/v3"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var dryRun bool

// main
func main() {
	app := cli.Command{
		Name:        "plax",
		Usage:       "Plex Local Asset eXporter",
		Description: "Export posters and background assets from Plex",
		Version:     "0.1.0",
		Authors:     []any{"Jeff Bailey <jeff@blahspam.com>"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Usage:    "Plex base URL",
				Value:    "http://127.0.0.1:32400",
				Sources:  cli.EnvVars("PLEX_URL"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "Plex authentication token",
				Sources:  cli.EnvVars("PLEX_TOKEN"),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "library",
				Aliases: []string{"l"},
				Usage:   "export assets from the named library",
				Value:   "all",
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "print assets to be exported without exporting them",
				Destination: &dryRun,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cl, err := plex.New(strings.TrimSuffix(cmd.String("url"), "/"), cmd.String("token"))
			if err != nil {
				return cli.Exit(err, -1)
			}

			// get available library
			libs, err := cl.Libraries(cmd.String("library"))
			if err != nil {
				return cli.Exit(err, -1)
			}
			if len(libs) == 0 {
				return cli.Exit("no libraries to export", 0)
			}

			// process each lib in parallel
			var wg sync.WaitGroup
			wg.Add(len(libs))

			// progress bars
			prog := mpb.New(mpb.WithWidth(64))
			defer prog.Wait()
			log.SetOutput(prog)

			defer prog.Wait()
			for i := range libs {
				bar := prog.AddBar(
					0,
					mpb.PrependDecorators(
						decor.Name(libs[i].Title, decor.WCSyncWidth),
						decor.CountersNoUnit(" %d/%d", decor.WCSyncWidth),
					),
					mpb.AppendDecorators(
						decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncWidthR),
						decor.Any(func(s decor.Statistics) string {
							if s.Total == 0 {
								return " preparing"
							}
							if s.Completed {
								return " done"
							}
							return ""
						}, decor.WCSyncWidthR),
					),
				)

				go func(cl *plex.Client, lib *plex.Library, bar *mpb.Bar) {
					defer wg.Done()
					defer bar.SetTotal(-1, true)

					// retrieving contents and update bar total
					contents, err := cl.Contents(lib)
					if err != nil {
						slog.Error("Error retrieving contents", slog.String("type", lib.Type), slog.String("err", err.Error()))
						return
					}
					bar.SetTotal(int64(len(contents)), false)

					for i := range contents {
						if err := cl.Download(&contents[i], cmd.Bool("dry-run")); err != nil {
							slog.Error("Error downloading content", slog.String("title", contents[i].Title), slog.String("err", err.Error()))
							continue
						}
						bar.Increment()
					}
				}(cl, &libs[i], bar)
			}

			wg.Wait()

			return nil
		},
	}

	ctx := context.Background()
	if err := app.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, err.Error())
	}
}
