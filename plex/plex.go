// Package plex provides a simplified Plex client.
package plex

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gpc "github.com/jrudio/go-plex-client"
)

// New constructs a new Plex client for the given URL and token.
func New(url string, token string) (*Client, error) {
	p, err := gpc.New(url, token)
	if err != nil {
		return nil, err
	}
	return &Client{p}, nil
}

// Client is wraps gpc.Plex with additional functionality.
type Client struct {
	plex *gpc.Plex
}

// Libraries returns the matching Libraries for the filter specified.
func (cl *Client) Libraries(filter string) ([]Library, error) {
	pls, err := cl.plex.GetLibraries()
	if err != nil {
		return nil, err
	}

	var libs []Library
	for _, libMeta := range pls.MediaContainer.Directory {
		if filter == "" || filter == "all" || filter == libMeta.Title {
			libs = append(libs, Library{
				Title: libMeta.Title,
				Key:   libMeta.Key,
				Type:  libMeta.Type,
			})
		}
	}

	return libs, nil
}

// Contents returns the Contents for the given Library.
func (cl *Client) Contents(l *Library) ([]Content, error) {
	switch l.Type {
	case "movie":
		return cl.movieContents(l)
	case "show":
		return cl.showContents(l)
	case "artist":
		return cl.artistContents(l)
	default:
		return nil, fmt.Errorf("unsupported library type: %s", l.Type)
	}
}

// Download assets for the supplied Content,
func (cl *Client) Download(c *Content, dryRun bool) error {
	if c.Art != "" {
		if err := cl.download(c.Art, c.Directory+"/background.jpg", dryRun); err != nil {
			return err
		}
	}
	if c.Thumb != "" {
		switch c.Type {
		case "album":
			if err := cl.download(c.Thumb, c.Directory+"/cover.jpg", dryRun); err != nil {
				return err
			}
		default:
			if err := cl.download(c.Thumb, c.Directory+"/poster.jpg", dryRun); err != nil {
				return err
			}
		}
	}
	return nil
}

// artistContents returns a Content for every artist and album.
func (cl *Client) artistContents(l *Library) ([]Content, error) {
	artists, err := cl.plex.GetLibraryContent(l.Key, "")
	if err != nil {
		return nil, fmt.Errorf("error retrieving contents: %w", err)
	}

	var contents []Content
	for _, artistMeta := range artists.MediaContainer.Metadata {
		var artistDir string

		// albums
		albums, err := cl.plex.GetEpisodes(artistMeta.RatingKey)
		if err != nil {
			return nil, fmt.Errorf("error retrieving %s albums: %w", artistMeta.Title, err)
		}
		for _, albumMeta := range albums.MediaContainer.Metadata {
			var albumDir string

			// tracks used for album & artist directories
			tracks, err := cl.plex.GetEpisodes(albumMeta.RatingKey)
			if err != nil {
				return nil, fmt.Errorf("error retrieving %s tracks: %w", albumMeta.Title, err)
			}
			for _, trackMeta := range tracks.MediaContainer.Metadata {
				albumDir = dir(trackMeta.Media)
				if albumDir != "" {
					if artistDir == "" {
						artistDir = filepath.Dir(albumDir)
					}
				}
			}

			// add album content
			contents = append(contents, Content{
				Title:     albumMeta.Title,
				Type:      albumMeta.Type,
				Directory: albumDir,
				Thumb:     albumMeta.Thumb,
				Art:       albumMeta.Art,
			})
		}

		// add artist content
		contents = append(contents, Content{
			Title:     artistMeta.Title,
			Type:      artistMeta.Type,
			Directory: artistDir,
			Thumb:     artistMeta.Thumb,
			Art:       artistMeta.Art,
		})
	}

	return contents, nil
}

// movieContents returns a Content for every movie.
func (cl *Client) movieContents(l *Library) ([]Content, error) {
	res, err := cl.plex.GetLibraryContent(l.Key, "")
	if err != nil {
		return nil, fmt.Errorf("error retrieving contents: %w", err)
	}

	var contents []Content
	for _, movieMeta := range res.MediaContainer.Metadata {
		contents = append(contents, Content{
			Title:     movieMeta.Title,
			Type:      movieMeta.Type,
			Directory: dir(movieMeta.Media),
			Thumb:     movieMeta.Thumb,
			Art:       movieMeta.Art,
		})
	}

	return contents, nil
}

// showContents returns a Content for every show and season.
func (cl *Client) showContents(l *Library) ([]Content, error) {
	if cl.plex == nil {
		return nil, fmt.Errorf("plex plex not set")
	}

	// shows
	shows, err := cl.plex.GetLibraryContent(l.Key, "")
	if err != nil {
		return nil, fmt.Errorf("error retrieving contents: %w", err)
	}

	var contents []Content
	for _, showMeta := range shows.MediaContainer.Metadata {
		var showDir string

		// seasons
		seasons, err := cl.plex.GetEpisodes(showMeta.RatingKey)
		if err != nil {
			return nil, fmt.Errorf("error retrieving %s seasons: %w", showMeta.Title, err)
		}
		for _, seasonMeta := range seasons.MediaContainer.Metadata {
			var seasonDir string

			// episode used for season & show directories
			episodes, err := cl.plex.GetEpisodes(seasonMeta.RatingKey)
			if err != nil {
				return nil, fmt.Errorf("error retrieving %s episodes: %w", showMeta.Title, err)
			}
			for _, episodeMeta := range episodes.MediaContainer.Metadata {
				seasonDir = dir(episodeMeta.Media)
				if seasonDir != "" {
					if showDir == "" {
						showDir = filepath.Dir(seasonDir)
					}
					break
				}
			}

			// add season contents
			contents = append(contents, Content{
				Title:     seasonMeta.Title,
				Type:      seasonMeta.Type,
				Directory: seasonDir,
				Thumb:     seasonMeta.Thumb,
				Art:       seasonMeta.Art,
			})
		}

		// add show contents
		contents = append(contents, Content{
			Title:     showMeta.Title,
			Type:      showMeta.Type,
			Directory: showDir,
			Thumb:     showMeta.Thumb,
			Art:       showMeta.Art,
		})
	}

	return contents, nil
}

// download the given asset path to the file specified
func (cl *Client) download(path string, file string, dryRun bool) error {
	// perform the request
	url := cl.plex.URL + path

	// if dry run, write the output to stdout
	if dryRun {
		log.Printf("Saving %s to %s\n", url, file)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error building request: %w", err)
	}
	req.Header.Add("X-Plex-Token", cl.plex.Token)

	resp, err := cl.plex.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing GET %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	// create the dir, if necessary (only useful when performing a test run)
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return fmt.Errorf("error creating %s: %w", filepath.Dir(file), err)
	}

	// create the file
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("error creating %s: %w", file, err)
	}
	defer f.Close()

	_, _ = io.Copy(f, resp.Body)
	return nil
}

// Library is a minimal struct modeling a Plex Library
type Library struct {
	Title string
	Key   string
	Type  string
}

// Content is a minimal struct modeling Plex Content.
type Content struct {
	Title     string
	Type      string
	Directory string
	Thumb     string
	Art       string
}

// dir returns the dir containing a slice of plex.Media.
func dir(med []gpc.Media) string {
	for _, m := range med {
		for _, p := range m.Part {
			if p.File != "" {
				return filepath.Dir(p.File)
			}
		}
	}
	return ""
}
