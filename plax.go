// Package main is the entrypoint for the plax application.
package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/blahspam/plax/plex"
	"github.com/urfave/cli/v2"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var dryRun bool

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
		Action: func(ctx *cli.Context) error {
			cl, err := plex.New(strings.TrimSuffix(ctx.String("url"), "/"), ctx.String("token"))
			if err != nil {
				return cli.Exit(err, -1)
			}

			// get available library
			libs, err := cl.Libraries(ctx.String("library"))
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
						log.Printf("error retrieving contents for %s: %s\n", lib.Type, err)
						return
					}
					bar.SetTotal(int64(len(contents)), false)

					for i := range contents {
						if err := cl.Download(&contents[i], ctx.Bool("dry-run")); err != nil {
							log.Printf("error downloading content for %s: %s\n", contents[i].Title, err)
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

	if err := app.Run(os.Args); err != nil {
		slog.Error(err.Error())
	}
}
